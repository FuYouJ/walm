package impl

import (
	"WarpCloud/walm/pkg/helm/impl/plugins"
	"WarpCloud/walm/pkg/k8s"
	k8sHelm "WarpCloud/walm/pkg/k8s/client/helm"
	"WarpCloud/walm/pkg/models/common"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/redis"
	"WarpCloud/walm/pkg/setting"
	"WarpCloud/walm/pkg/util"
	"WarpCloud/walm/pkg/util/transwarpjsonnet"
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"helm.sh/helm/pkg/action"
	"helm.sh/helm/pkg/chart"
	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/pkg/kube"
	"helm.sh/helm/pkg/registry"
	helmRelease "helm.sh/helm/pkg/release"
	"helm.sh/helm/pkg/storage"
	"helm.sh/helm/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"net/http"
	"os"
	"strings"
)

const (
	compatibleNamespace = "kube-system"
)

type ChartRepository struct {
	Name     string
	URL      string
	Username string
	Password string
}

type Helm struct {
	chartRepoMap   map[string]*ChartRepository
	registryClient *registry.Client
	k8sCache       k8s.Cache
	list           *action.List
	kubeClients    *k8sHelm.Client

	actionConfigs *lru.Cache
}

func (helmImpl *Helm) getActionConfig(namespace string) (*action.Configuration, error) {
	if actionConfig, ok := helmImpl.actionConfigs.Get(namespace); ok {
		return actionConfig.(*action.Configuration), nil
	} else {
		kubeConfig, kubeClient := helmImpl.kubeClients.GetKubeClient(namespace)
		clientset, err := kubeClient.Factory.KubernetesClientSet()
		if err != nil {
			klog.Errorf("failed to get clientset: %s", err.Error())
			return nil, err
		}

		d := driver.NewConfigMapsEx(clientset.CoreV1().ConfigMaps(namespace), clientset.CoreV1().ConfigMaps(compatibleNamespace), namespace)
		store := storage.Init(d)
		config := &action.Configuration{
			KubeClient:       kubeClient,
			Releases:         store,
			RESTClientGetter: kubeConfig,
			Log:              klog.Infof,
		}
		helmImpl.actionConfigs.Add(namespace, config)
		return config, nil
	}
}

func (helmImpl *Helm) ListAllReleases() (releaseCaches []*release.ReleaseCache, err error) {
	helmReleases, err := helmImpl.list.Run()
	if err != nil {
		klog.Errorf("failed to list helm releases: %s\n", err.Error())
		return nil, err
	}

	filteredHelmReleases := filterHelmReleases(helmReleases)
	for _, helmRelease := range filteredHelmReleases {
		releaseCache, err := helmImpl.convertHelmRelease(helmRelease)
		if err != nil {
			klog.Errorf("failed to convert helm release %s/%s : %s", helmRelease.Namespace, helmRelease.Name, err.Error())
			return nil, err
		}
		releaseCaches = append(releaseCaches, releaseCache)
	}
	return
}

// keep latest deployed one. if there is no deployed one ,keep the latest version.
func filterHelmReleases(releases []*helmRelease.Release) (filteredReleases map[string]*helmRelease.Release) {
	filteredReleases = map[string]*helmRelease.Release{}
	for _, release := range releases {
		filedName := redis.BuildFieldName(release.Namespace, release.Name)
		if existedRelease, ok := filteredReleases[filedName]; ok {
			if existedRelease.Info != nil && existedRelease.Info.Status == helmRelease.StatusDeployed {
				if release.Info != nil && release.Info.Status == helmRelease.StatusDeployed &&
					existedRelease.Version < release.Version {
					filteredReleases[filedName] = release
				}
			} else {
				if release.Info != nil && release.Info.Status == helmRelease.StatusDeployed {
					filteredReleases[filedName] = release
				} else if existedRelease.Version < release.Version {
					filteredReleases[filedName] = release
				}
			}
		} else {
			filteredReleases[filedName] = release
		}
	}
	return
}

func (helmImpl *Helm) DeleteRelease(namespace string, name string) error {
	action, err := helmImpl.getDeleteAction(namespace)
	if err != nil {
		klog.Errorf("failed to get current helm client : %s", err.Error())
		return err
	}

	_, err = action.Run(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			klog.Warningf("release %s is not found from helm", name)
		} else {
			klog.Errorf("failed to delete release from helm : %s", err.Error())
			return err
		}
	}
	return nil
}

func (helmImpl *Helm) InstallOrCreateRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile,
	dryRun bool, update bool, oldReleaseInfo *release.ReleaseInfoV2, paused *bool) (*release.ReleaseCache, error) {
	var rawChart *chart.Chart
	var chartErr error
	// priority: chartFiles > chartImage > chartName
	if chartFiles != nil {
		rawChart, chartErr = loader.LoadFiles(convertBufferFiles(chartFiles))
	} else if releaseRequest.ChartImage != "" {
		rawChart, chartErr = helmImpl.getRawChartByImage(releaseRequest.ChartImage)
	} else {
		rawChart, chartErr = helmImpl.getRawChartFromRepo(releaseRequest.RepoName, releaseRequest.ChartName, releaseRequest.ChartVersion)
	}
	if chartErr != nil {
		klog.Errorf("failed to get raw chart : %s", chartErr.Error())
		return nil, chartErr
	}

	chartInfo, err := buildChartInfo(rawChart)
	if err != nil {
		klog.Errorf("failed to build chart info : %s", err.Error())
		return nil, err
	}

	if releaseRequest.ConfigValues == nil {
		releaseRequest.ConfigValues = map[string]interface{}{}
	}

	if chartInfo.WalmVersion == common.WalmVersionV2 {
		// support meta pretty parameters
		if releaseRequest.MetaInfoParams != nil {
			metaInfoConfigs, err := releaseRequest.MetaInfoParams.BuildConfigValues(chartInfo.MetaInfo)
			if err != nil {
				klog.Errorf("failed to get meta info parameters : %s", err.Error())
				return nil, err
			}
			util.MergeValues(releaseRequest.ConfigValues, metaInfoConfigs, false)
		}
	} else if chartInfo.WalmVersion == common.WalmVersionV1 {
		// compatible for v1 pretty params
		if releaseRequest.ReleasePrettyParams != nil {
			processPrettyParams(&releaseRequest.ReleaseRequest)
		}
	}

	dependencies := releaseRequest.Dependencies
	releaseLabels := releaseRequest.ReleaseLabels
	releasePlugins := releaseRequest.Plugins
	configValues := releaseRequest.ConfigValues
	if update {
		// reuse config values, dependencies, release labels, walm plugins
		configValues, dependencies, releaseLabels, releasePlugins, err = reuseReleaseRequest(oldReleaseInfo, releaseRequest)
		if err != nil {
			klog.Errorf("failed to reuse release request : %s", err.Error())
			return nil, err
		}
	}

	if chartInfo.MetaInfo != nil {
		releasePlugins, err = mergeReleasePlugins(releasePlugins, chartInfo.MetaInfo.Plugins)
		if err != nil {
			klog.Errorf("failed to merge chart default plugins : %s", err.Error())
			return nil, err
		}
	}

	// get all the dependency releases' output configs from ReleaseConfig or dummy service(for compatible)
	dependencyConfigs, err := helmImpl.GetDependencyOutputConfigs(namespace, dependencies, chartInfo, true)
	if err != nil {
		klog.Errorf("failed to get all the dependency releases' output configs : %s", err.Error())
		return nil, err
	}

	if chartInfo.WalmVersion == common.WalmVersionV2 {
		err = transwarpjsonnet.ProcessJsonnetChart(
			releaseRequest.RepoName, rawChart, namespace,
			releaseRequest.Name, configValues, dependencyConfigs,
			dependencies, releaseLabels, releaseRequest.ChartImage,
		)
		if err != nil {
			klog.Errorf("failed to ProcessJsonnetChart : %s", err.Error())
			return nil, err
		}
	} else if chartInfo.WalmVersion == common.WalmVersionV1 {
		err = transwarpjsonnet.ProcessJsonnetChartV1(
			releaseRequest.RepoName, rawChart, namespace,
			releaseRequest.Name, configValues, dependencyConfigs,
			dependencies, releaseLabels, releaseRequest.ChartImage,
		)
		if err != nil {
			klog.Errorf("failed to ProcessJsonnetChart v1: %s", err.Error())
			return nil, err
		}
	}

	if paused != nil {
		if *paused {
			releasePlugins, err = mergeReleasePlugins([]*release.ReleasePlugin{
				{
					Name:    plugins.PauseReleasePluginName,
					Version: "1.0",
				},
			}, releasePlugins)
		} else {
			releasePlugins, err = mergeReleasePlugins([]*release.ReleasePlugin{
				{
					Name:    plugins.PauseReleasePluginName,
					Version: "1.0",
					Disable: true,
				},
			}, releasePlugins)
		}
	}
	// add default plugin
	releasePlugins = append(releasePlugins, &release.ReleasePlugin{
		Name: plugins.ValidateReleaseConfigPluginName,
	})

	valueOverride := map[string]interface{}{}
	util.MergeValues(valueOverride, dependencyConfigs, false)
	util.MergeValues(valueOverride, configValues, false)

	valueOverride[plugins.WalmPluginConfigKey] = releasePlugins
	releaseCache, err := helmImpl.doInstallUpgradeReleaseFromChart(namespace, releaseRequest, rawChart, valueOverride, update, dryRun, releasePlugins)
	if err != nil {
		klog.Errorf("failed to create or update release from chart : %s", err.Error())
		return nil, err
	}

	return releaseCache, nil
}

func (helmImpl *Helm) doInstallUpgradeReleaseFromChart(namespace string,
	releaseRequest *release.ReleaseRequestV2, rawChart *chart.Chart, valueOverride map[string]interface{},
	update bool, dryRun bool, releasePlugins []*release.ReleasePlugin) (releaseCache *release.ReleaseCache, err error) {

	releaseChan := make(chan *helmRelease.Release, 1)
	releaseErrChan := make(chan error, 1)

	expChan := make(chan struct{})
	_, kubeClient := helmImpl.kubeClients.GetKubeClient(namespace)

	// execute pre_install plugins
	go func() {
		select {
		case release := <-releaseChan:
			defer func() {
				if err := recover(); err != nil {
					releaseErrChan <- errors.New(fmt.Sprintf("panic happend: %v", err))
				}
			}()
			context, err := buildContext(kubeClient, release)
			if err != nil {
				releaseErrChan <- err
				return
			}

			err = runPlugins(releasePlugins, context, plugins.Pre_Install)
			if err != nil {
				releaseErrChan <- err
				return
			}

			manifest, err := buildManifest(context.Resources)
			if err != nil {
				klog.Errorf("failed to build manifest : %s", err.Error())
				releaseErrChan <- err
				return
			}
			release.Manifest = manifest
			releaseChan <- release
		case <-expChan:
			klog.Warning("failed to execute pre_install plugins with exception")
		}

	}()
	defer close(expChan)

	var helmRelease *helmRelease.Release
	if update {
		action, err := helmImpl.getUpgradeAction(namespace)
		if err != nil {
			return nil, err
		}
		action.DryRun = dryRun
		action.Namespace = namespace
		action.MaxHistory = 3
		action.ReleaseChan = releaseChan
		action.ReleaseErrChan = releaseErrChan
		helmRelease, err = action.Run(releaseRequest.Name, rawChart, valueOverride)
		if err != nil {
			klog.Errorf("failed to upgrade release %s/%s from chart : %s", namespace, releaseRequest.Name, err.Error())
			return nil, err
		}
	} else {
		action, err := helmImpl.getInstallAction(namespace)
		if err != nil {
			return nil, err
		}
		action.DryRun = dryRun
		action.Namespace = namespace
		action.ReleaseName = releaseRequest.Name
		action.ReleaseChan = releaseChan
		action.ReleaseErrChan = releaseErrChan
		helmRelease, err = action.Run(rawChart, valueOverride)
		if err != nil {
			klog.Errorf("failed to install release %s/%s from chart : %s", namespace, releaseRequest.Name, err.Error())
			if !dryRun {
				action1, err1 := helmImpl.getDeleteAction(namespace)
				if err1 != nil {
					klog.Errorf("failed to get helm delete action : %s", err.Error())
				} else {
					_, err1 = action1.Run(releaseRequest.Name)
					if err1 != nil {
						klog.Errorf("failed to rollback to delete release %s/%s : %s", namespace, releaseRequest.Name, err1.Error())
					}
				}
			}
			return nil, err
		}
	}

	// execute post_install plugins
	context, err := buildContext(kubeClient, helmRelease)
	if err != nil {
		return nil, err
	}

	err = runPlugins(releasePlugins, context, plugins.Post_Install)
	if err != nil {
		return nil, err
	}
	return helmImpl.convertHelmRelease(helmRelease)
}

func buildContext(kubeClient *kube.Client, release *helmRelease.Release) (*plugins.PluginContext, error) {
	resources, err := kubeClient.Build(bytes.NewBufferString(release.Manifest))
	if err != nil {
		klog.Errorf("failed to build k8s resources : %s", err.Error())
		return nil, err
	}
	context := &plugins.PluginContext{
		R:         release,
		Resources: []runtime.Object{},
	}
	for _, resource := range resources {
		context.Resources = append(context.Resources, resource.Object)
	}
	return context, nil
}

func runPlugins(releasePlugins []*release.ReleasePlugin, context *plugins.PluginContext, runnerType plugins.RunnerType) error {
	for _, plugin := range releasePlugins {
		if plugin.Disable {
			continue
		}
		runner := plugins.GetRunner(plugin)
		if runner != nil && runner.Type == runnerType {
			klog.Infof("start to exec %s plugin %s", runnerType, plugin.Name)
			err := runner.Run(context, plugin.Args)
			if err != nil {
				klog.Errorf("failed to exec %s plugin %s : %s", runnerType, plugin.Name, err.Error())
				return err
			}
			klog.Infof("succeed to exec %s plugin %s", runnerType, plugin.Name)
		}
	}
	return nil
}

func buildManifest(resources []runtime.Object) (string, error) {
	var sb strings.Builder
	for _, resource := range resources {
		resourceBytes, err := yaml.Marshal(resource)
		if err != nil {
			return "", err
		}
		sb.WriteString("\n---\n")
		sb.Write(resourceBytes)
	}
	return sb.String(), nil
}

func (helmImpl *Helm) convertHelmRelease(helmRelease *helmRelease.Release) (releaseCache *release.ReleaseCache, err error) {
	releaseSpec := release.ReleaseSpec{}
	releaseSpec.Name = helmRelease.Name
	releaseSpec.Namespace = helmRelease.Namespace
	releaseSpec.Dependencies = make(map[string]string)
	releaseSpec.Version = int32(helmRelease.Version)
	releaseSpec.ChartVersion = helmRelease.Chart.Metadata.Version
	releaseSpec.ChartName = helmRelease.Chart.Metadata.Name
	releaseSpec.ChartAppVersion = helmRelease.Chart.Metadata.AppVersion
	releaseSpec.ConfigValues = map[string]interface{}{}
	util.MergeValues(releaseSpec.ConfigValues, helmRelease.Config, false)
	releaseCache = &release.ReleaseCache{
		ReleaseSpec: releaseSpec,
	}

	releaseCache.ComputedValues, err = chartutil.CoalesceValues(helmRelease.Chart, helmRelease.Config)
	if err != nil {
		klog.Errorf("failed to get computed values : %s", err.Error())
		return nil, err
	}

	releaseCache.MetaInfoValues, _ = buildMetaInfoValues(helmRelease.Chart, releaseCache.ComputedValues)
	releaseCache.ReleaseResourceMetas, err = helmImpl.getReleaseResourceMetas(helmRelease)
	if err != nil {
		return nil, err
	}
	releaseCache.Manifest = helmRelease.Manifest
	releaseCache.HelmVersion = helmRelease.HelmVersion
	return
}

func (helmImpl *Helm) getReleaseResourceMetas(helmRelease *helmRelease.Release) (resources []release.ReleaseResourceMeta, err error) {
	resources = []release.ReleaseResourceMeta{}
	_, kubeClient := helmImpl.kubeClients.GetKubeClient(helmRelease.Namespace)
	results, err := kubeClient.Build(bytes.NewBufferString(helmRelease.Manifest))
	if err != nil {
		klog.Errorf("failed to get release resource metas of %s", helmRelease.Name)
		return resources, err
	}
	for _, result := range results {
		resource := release.ReleaseResourceMeta{
			Kind:      k8sModel.ResourceKind(result.Object.GetObjectKind().GroupVersionKind().Kind),
			Namespace: result.Namespace,
			Name:      result.Name,
		}
		resources = append(resources, resource)
	}
	return
}

func buildMetaInfoValues(chart *chart.Chart, computedValues map[string]interface{}) (*release.MetaInfoParams, error) {
	chartMetaInfo, err := getChartMetaInfo(chart)
	if err != nil {
		return nil, err
	}
	if chartMetaInfo != nil {
		metaInfoParams, err := chartMetaInfo.BuildMetaInfoParams(computedValues)
		if err != nil {
			return nil, err
		}
		return metaInfoParams, nil
	}

	return nil, nil
}

func (helmImpl *Helm) getInstallAction(namespace string) (*action.Install, error) {
	config, err := helmImpl.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	return action.NewInstall(config), nil
}

func (helmImpl *Helm) getUpgradeAction(namespace string) (*action.Upgrade, error) {
	config, err := helmImpl.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	return action.NewUpgrade(config), nil
}

func (helmImpl *Helm) getDeleteAction(namespace string) (*action.Uninstall, error) {
	config, err := helmImpl.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	return action.NewUninstall(config), nil
}

func reuseReleaseRequest(releaseInfo *release.ReleaseInfoV2, releaseRequest *release.ReleaseRequestV2) (
	configValues map[string]interface{}, dependencies map[string]string, releaseLabels map[string]string, walmPlugins []*release.ReleasePlugin, err error) {

	configValues = map[string]interface{}{}
	util.MergeValues(configValues, releaseInfo.ConfigValues, false)
	util.MergeValues(configValues, releaseRequest.ConfigValues, false)

	dependencies = map[string]string{}
	for key, value := range releaseInfo.Dependencies {
		dependencies[key] = value
	}
	for key, value := range releaseRequest.Dependencies {
		if value == "" {
			if _, ok := dependencies[key]; ok {
				delete(dependencies, key)
			}
		} else {
			dependencies[key] = value
		}
	}

	releaseLabels = map[string]string{}
	for key, value := range releaseInfo.ReleaseLabels {
		releaseLabels[key] = value
	}
	for key, value := range releaseRequest.ReleaseLabels {
		if value == "" {
			if _, ok := releaseLabels[key]; ok {
				delete(releaseLabels, key)
			}
		} else {
			releaseLabels[key] = value
		}
	}

	walmPlugins, err = mergeReleasePlugins(releaseRequest.Plugins, releaseInfo.Plugins)
	if err != nil {
		return
	}
	return
}

func mergeReleasePlugins(plugins, defaultPlugins []*release.ReleasePlugin) (mergedPlugins []*release.ReleasePlugin, err error) {
	releasePluginsMap := map[string]*release.ReleasePlugin{}
	for _, plugin := range plugins {
		if _, ok := releasePluginsMap[plugin.Name]; ok {
			return nil, buildDuplicatedPluginError(plugin.Name)
		} else {
			releasePluginsMap[plugin.Name] = plugin
		}
	}
	for _, plugin := range defaultPlugins {
		if _, ok := releasePluginsMap[plugin.Name]; !ok {
			releasePluginsMap[plugin.Name] = plugin
		}
	}
	for _, plugin := range releasePluginsMap {
		mergedPlugins = append(mergedPlugins, plugin)
	}
	return
}

func buildDuplicatedPluginError(pluginName string) error {
	return fmt.Errorf("more than one plugin %s is not allowed", pluginName)
}

func convertBufferFiles(chartFiles []*common.BufferedFile) []*loader.BufferedFile {
	result := []*loader.BufferedFile{}
	for _, file := range chartFiles {
		result = append(result, &loader.BufferedFile{
			Name: file.Name,
			Data: file.Data,
		})
	}
	return result
}

func NewHelm(repoList []*setting.ChartRepo, registryClient *registry.Client, k8sCache k8s.Cache, kubeClients *k8sHelm.Client) (*Helm, error) {
	chartRepoMap := make(map[string]*ChartRepository)

	for _, chartRepo := range repoList {
		chartRepository := ChartRepository{
			Name:     chartRepo.Name,
			URL:      chartRepo.URL,
			Username: "",
			Password: "",
		}
		chartRepoMap[chartRepo.Name] = &chartRepository
	}

	actionConfigs, _ := lru.New(100)

	helm := &Helm{
		k8sCache:       k8sCache,
		kubeClients:    kubeClients,
		registryClient: registryClient,
		chartRepoMap:   chartRepoMap,
		actionConfigs:  actionConfigs,
	}

	actionConfig, err := helm.getActionConfig("")
	if err != nil {
		return nil, err
	}
	list := action.NewList(actionConfig)
	list.AllNamespaces = true
	list.All = true
	list.StateMask = action.ListDeployed | action.ListFailed | action.ListPendingInstall | action.ListPendingRollback |
		action.ListPendingUpgrade | action.ListUninstalled | action.ListUninstalling | action.ListUnknown

	helm.list = list

	return helm, nil

}

func NewRegistryClient(chartImageConfig *setting.ChartImageConfig) (*registry.Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	option := &registry.ClientOptions{
		Out: os.Stdout,
		Resolver: docker.NewResolver(docker.ResolverOptions{
			Client: client,
		}),
	}

	return registry.NewClient(option)
}
