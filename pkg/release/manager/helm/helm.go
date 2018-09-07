package helm

import (
	"fmt"
	"io/ioutil"
	"strings"

	"walm/pkg/setting"
	"walm/pkg/release"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/storage/driver"
	"k8s.io/helm/pkg/transwarp"
	"sync"
	"errors"
	"walm/pkg/release/manager/helm/cache"
	"time"
	"walm/pkg/redis"
	"walm/pkg/k8s/client"
	"k8s.io/apimachinery/pkg/util/wait"
	hapiRelease "k8s.io/helm/pkg/proto/hapi/release"
	walmerr "walm/pkg/util/error"
)

const (
	helmCacheDefaultResyncInterval time.Duration = 5 * time.Minute
)

type ChartRepository struct {
	Name     string
	URL      string
	Username string
	Password string
}

type HelmClient struct {
	client                  *helm.Client
	chartRepoMap            map[string]*ChartRepository
	dryRun                  bool
	helmCache               *cache.HelmCache
	helmCacheResyncInterval time.Duration
}

var helmClient *HelmClient

func GetDefaultHelmClient() *HelmClient {
	return helmClient
}

func InitHelm() {
	tillerHost := setting.Config.SysHelm.TillerHost
	client1 := helm.NewClient(helm.Host(tillerHost))
	chartRepoMap := make(map[string]*ChartRepository)

	for _, chartRepo := range setting.Config.RepoList {
		chartRepository := ChartRepository{
			Name:     chartRepo.Name,
			URL:      chartRepo.URL,
			Username: "",
			Password: "",
		}
		chartRepoMap[chartRepo.Name] = &chartRepository
	}

	helmCache := cache.NewHelmCache(redis.GetDefaultRedisClient(), client1, client.GetKubeClient())

	helmClient = &HelmClient{
		client:                  client1,
		chartRepoMap:            chartRepoMap,
		dryRun:                  false,
		helmCache:               helmCache,
		helmCacheResyncInterval: helmCacheDefaultResyncInterval,
	}
}

func InitHelmByParams(tillerHost string, chartRepoMap map[string]*ChartRepository, dryRun bool) {
	client := helm.NewClient(helm.Host(tillerHost))

	helmClient = &HelmClient{
		client:       client,
		chartRepoMap: chartRepoMap,
		dryRun:       dryRun,
	}
}

func (client *HelmClient) ListReleases(namespace, filter string) ([]*release.ReleaseInfo, error) {
	logrus.Debugf("Enter ListReleases namespace=%s filter=%s\n", namespace, filter)
	releaseCaches, err := client.helmCache.GetReleaseCaches(namespace, filter, 0)
	if err != nil {
		logrus.Errorf("failed to get release caches with namespace=%s filter=%s : %s", namespace, filter, err.Error())
		return nil, err
	}

	releaseInfos := []*release.ReleaseInfo{}
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	for _, releaseCache := range releaseCaches {
		wg.Add(1)
		go func(releaseCache *release.ReleaseCache) {
			defer wg.Done()
			info, err1 := buildReleaseInfo(releaseCache)
			if err1 != nil {
				err = errors.New(fmt.Sprintf("failed to build release info: %s\n", err1.Error()))
				logrus.Error(err.Error())
				return
			}
			mux.Lock()
			releaseInfos = append(releaseInfos, info)
			mux.Unlock()
		}(releaseCache)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	return releaseInfos, nil
}

func (client *HelmClient) GetRelease(namespace, releaseName string) (release *release.ReleaseInfo, err error) {
	logrus.Debugf("Enter GetRelease %s %s\n", namespace, releaseName)
	releaseCache, err := client.helmCache.GetReleaseCache(namespace, releaseName)
	if err != nil {
		logrus.Errorf("failed to get release cache of %s : %s", releaseName, err.Error())
		return nil, err
	}
	release, err = buildReleaseInfo(releaseCache)
	if err != nil {
		logrus.Errorf("failed to build release info: %s\n", err.Error())
		return
	}
	return
}

func (client *HelmClient) InstallUpgradeRealese(namespace string, releaseRequest *release.ReleaseRequest) error {
	if releaseRequest.ConfigValues == nil {
		releaseRequest.ConfigValues = map[string]interface{}{}
	}
	if releaseRequest.Dependencies == nil {
		releaseRequest.Dependencies = map[string]string{}
	}
	chartRequested, err := client.getChartRequest(releaseRequest.RepoName, releaseRequest.ChartName, releaseRequest.ChartVersion)
	if err != nil {
		logrus.Errorf("failed to get chart %s/%s:%s", releaseRequest.RepoName, releaseRequest.ChartName, releaseRequest.ChartVersion)
		return err
	}
	depLinks := make(map[string]interface{})
	for k, v := range releaseRequest.Dependencies {
		depLinks[k] = v
	}
	helmRelease, err := client.installChart(releaseRequest.Name, namespace, releaseRequest.ConfigValues, depLinks, chartRequested)
	if err != nil {
		logrus.Errorf("failed to install chart : %s", err.Error())
		return err
	}

	err = client.helmCache.CreateOrUpdateReleaseCache(helmRelease)
	if err != nil {
		logrus.Errorf("failed to create of update release cache of %s : %s", helmRelease.Name, err.Error())
		// TODO rollback helm release
		return err
	}
	logrus.Infof("succeed to create or update release %s", releaseRequest.Name)
	return nil
}

func (client *HelmClient) RollbackRealese(namespace, releaseName, version string) error {
	return nil
}

func (client *HelmClient) DeleteRelease(namespace, releaseName string) error {
	logrus.Debugf("Enter DeleteRelease %s %s\n", namespace, releaseName)

	_, err := client.GetRelease(namespace, releaseName)
	if err != nil {
		if walmerr.IsNotFoundError(err) {
			logrus.Warnf("release %s is not found", releaseName)
			return nil
		}
		logrus.Errorf("failed to get release %s : %s", releaseName, err.Error())
		return err
	}

	opts := []helm.DeleteOption{
		helm.DeletePurge(true),
	}
	res, err := client.client.DeleteRelease(
		releaseName, opts...,
	)
	if err != nil {
		logrus.Errorf("failed to delete release : %s", err.Error())
		return err
	}
	if res != nil && res.Info != "" {
		logrus.Println(res.Info)
	}

	err = client.helmCache.DeleteReleaseCache(namespace, releaseName)
	if err != nil {
		logrus.Errorf("failed to delete release cache of %s : %s", releaseName, err.Error())
		//TODO rollback?
		return err
	}

	logrus.Infof("DeleteRelease %s %s Success\n", namespace, releaseName)

	return err
}

func (client *HelmClient) GetDependencies(repoName, chartName, chartVersion string) (subChartNames []string, err error) {
	logrus.Debugf("Enter GetDependencies %s %s\n", chartName, chartVersion)
	chartRequested, err := client.getChartRequest(repoName, chartName, chartVersion)
	if err != nil {
		return nil, err
	}
	dependencies, err := parseChartDependencies(chartRequested)
	if err != nil {
		return nil, err
	}
	return dependencies, nil
}

func (client *HelmClient) downloadChart(repoName, charName, version string) (string, error) {
	if repoName == "" {
		repoName = "stable"
	}
	repo, ok := client.chartRepoMap[repoName]
	if !ok {
		return "", fmt.Errorf("can not find repo name: %s", repoName)
	}
	chartURL, httpGetter, err := FindChartInChartMuseumRepoURL(repo.URL, "", "", charName, version)
	if err != nil {
		return "", err
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	filename, err := ChartMuseumDownloadTo(chartURL, tmpDir, httpGetter)
	if err != nil {
		logrus.Printf("DownloadTo err %v", err)
		return "", err
	}

	return filename, nil
}

func (client *HelmClient) getChartRequest(repoName, chartName, chartVersion string) (*chart.Chart, error) {
	chartPath, err := client.downloadChart(repoName, chartName, chartVersion)
	if err != nil {
		logrus.Errorf("failed to download chart : %s", err.Error())
		return nil, err
	}
	chartRequested, err := chartutil.Load(chartPath)
	if err != nil {
		logrus.Errorf("failed to load chart : %s", err.Error())
		return nil, err
	}

	return chartRequested, nil
}

func (client *HelmClient) installChart(releaseName, namespace string, configValues map[string]interface{}, depLinks map[string]interface{}, chart *chart.Chart) (*hapiRelease.Release, error) {
	rawVals, err := yaml.Marshal(configValues)
	if err != nil {
		logrus.Errorf("failed to marshal config values: %s", err.Error())
		return nil, err
	}
	err = transwarp.ProcessAppCharts(client.client, chart, releaseName, namespace, string(rawVals[:]), depLinks)
	if err != nil {
		logrus.Errorf("failed to process app charts : %s", err.Error())
		return nil, err
	}

	helmRelease := &hapiRelease.Release{}
	releaseHistory, err := client.client.ReleaseHistory(releaseName, helm.WithMaxHistory(1))
	if err == nil {
		previousReleaseNamespace := releaseHistory.Releases[0].Namespace
		if previousReleaseNamespace != namespace {
			logrus.Warnf("namespace %s doesn't match with previous, release will be deployed to %s",
				namespace, previousReleaseNamespace,
			)
		}
		resp, err := client.client.UpdateReleaseFromChart(
			releaseName,
			chart,
			helm.UpdateValueOverrides(rawVals),
			helm.ReuseValues(true),
			helm.UpgradeDryRun(client.dryRun),
		)
		if err != nil {
			logrus.Errorf("failed to update release from chart : %s", err.Error())
			return nil, err
		}
		helmRelease = resp.GetRelease()
	} else if strings.Contains(err.Error(), driver.ErrReleaseNotFound(releaseName).Error()) {
		resp, err := client.client.InstallReleaseFromChart(
			chart,
			namespace,
			helm.ValueOverrides(rawVals),
			helm.ReleaseName(releaseName),
			helm.InstallDryRun(client.dryRun),
		)
		if err != nil {
			logrus.Errorf("failed to install release from chart : %s", err.Error())
			return nil, err
		}
		helmRelease = resp.GetRelease()
	} else {
		logrus.Errorf("failed to get release history : %s", err.Error())
		return nil, err
	}

	return helmRelease, nil
}

func (client *HelmClient) StartResyncReleaseCaches(stopCh <-chan struct{}) {
	logrus.Infof("start to resync release cache every %v", client.helmCacheResyncInterval)
	go wait.Until(func() {
		client.helmCache.Resync()
	}, client.helmCacheResyncInterval, stopCh)
}