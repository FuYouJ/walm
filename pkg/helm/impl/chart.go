package impl

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/util/transwarpjsonnet"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/go-resty/resty"
	"github.com/pkg/errors"
	"helm.sh/helm/pkg/chart"
	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/registry"
	"helm.sh/helm/pkg/repo"
	"io/ioutil"
	"k8s.io/klog"
	"net/url"
	"path/filepath"
	"strings"
	"WarpCloud/walm/pkg/models/common"
)

func (helmImpl *Helm) GetChartAutoDependencies(repoName, chartName, chartVersion string) (subChartNames []string, err error) {
	klog.V(2).Infof("Enter GetAutoDependencies %s %s\n", chartName, chartVersion)

	subChartNames = []string{}
	detailChartInfo, err := helmImpl.GetChartDetailInfo(repoName, chartName, chartVersion)
	if err != nil {
		return nil, err
	}

	if detailChartInfo.WalmVersion == common.WalmVersionV2 {
		if detailChartInfo.MetaInfo != nil && detailChartInfo.MetaInfo.ChartDependenciesInfo != nil {
			for _, dependency := range detailChartInfo.MetaInfo.ChartDependenciesInfo {
				if dependency.AutoDependency() {
					subChartNames = append(subChartNames, dependency.Name)
				}
			}
		}
	} else if detailChartInfo.WalmVersion == common.WalmVersionV1 {
		for _, dependencyChart := range detailChartInfo.DependencyCharts {
			subChartNames = append(subChartNames, dependencyChart.ChartName)
		}
	}

	return subChartNames, nil
}

func (helmImpl *Helm) GetRepoList() *release.RepoInfoList {
	repoInfoList := new(release.RepoInfoList)
	repoInfoList.Items = make([]*release.RepoInfo, 0)
	for _, v := range helmImpl.chartRepoMap {
		repoInfo := release.RepoInfo{}
		repoInfo.TenantRepoName = v.Name
		repoInfo.TenantRepoURL = v.URL
		repoInfoList.Items = append(repoInfoList.Items, &repoInfo)
	}
	return repoInfoList
}

func (helmImpl *Helm) GetChartDetailInfo(repoName, chartName, chartVersion string) (*release.ChartDetailInfo, error) {
	rawChart, err := helmImpl.getRawChartFromRepo(repoName, chartName, chartVersion)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			err = errorModel.NotFoundError{}
		}
		return nil, err
	}

	return buildChartInfo(rawChart)
}

func (helmImpl *Helm) GetChartList(repoName string) (*release.ChartInfoList, error) {
	chartInfoList := new(release.ChartInfoList)
	chartInfoList.Items = make([]*release.ChartInfo, 0)
	chartRepository, ok := helmImpl.chartRepoMap[repoName]
	if !ok {
		return nil, fmt.Errorf("can't find repo name %s", repoName)
	}
	indexFile, err := getChartIndexFile(chartRepository.URL, chartRepository.Username, chartRepository.Password)
	if err != nil {
		return nil, err
	}
	for _, cvs := range indexFile.Entries {
		for _, cv := range cvs {
			chartInfo := new(release.ChartInfo)
			chartInfo.ChartName = cv.Name
			chartInfo.ChartVersion = cv.Version
			chartInfo.ChartAppVersion = cv.AppVersion
			chartInfo.ChartEngine = "transwarp"
			chartInfo.ChartDescription = cv.Description
			chartInfoList.Items = append(chartInfoList.Items, chartInfo)
		}
	}
	return chartInfoList, nil
}

func (helmImpl *Helm) GetDetailChartInfoByImage(chartImage string) (*release.ChartDetailInfo, error) {
	rawChart, err := helmImpl.getRawChartByImage(chartImage)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			err = errorModel.NotFoundError{}
		}
		return nil, err
	}

	return buildChartInfo(rawChart)
}

func (helmImpl *Helm) getRawChartFromRepo(repoName, chartName, chartVersion string) (rawChart *chart.Chart, err error) {
	chartPath, err := helmImpl.downloadChartFromRepo(repoName, chartName, chartVersion)
	if err != nil {
		klog.Errorf("failed to download chart : %s", err.Error())
		return nil, err
	}

	chartLoader, err := loader.Loader(chartPath)
	if err != nil {
		klog.Errorf("failed to init chartLoader : %s", err.Error())
		return nil, err
	}

	return chartLoader.Load()
}

func (helmImpl *Helm) getRawChartByImage(chartImage string) (*chart.Chart, error) {
	ref, err := registry.ParseReference(chartImage)
	if err != nil {
		klog.Errorf("failed to parse chart image %s : %s", chartImage, err.Error())
		return nil, err
	}

	err = helmImpl.registryClient.PullChart(ref)
	if err != nil {
		klog.Errorf("failed to pull chart %s : %s", chartImage, err.Error())
		return nil, err
	}

	chart, err := helmImpl.registryClient.LoadChart(ref)
	if err != nil {
		klog.Errorf("failed to load chart %s : %s", chartImage, err.Error())
		return nil, err
	}
	return chart, nil
}

func getChartMetaInfo(rawChart *chart.Chart) (chartMetaInfo *release.ChartMetaInfo, err error) {
	for _, f := range rawChart.Files {
		if f.Name == transwarpjsonnet.TranswarpMetadataDir+transwarpjsonnet.TranswarpMetaInfoFileName {
			chartMetaInfo = &release.ChartMetaInfo{}
			err = yaml.Unmarshal(f.Data, chartMetaInfo)
			if err != nil {
				klog.Error(errors.Wrapf(err, "chart %s-%s MetaInfo Unmarshal metainfo.yaml error",
					rawChart.Metadata.Name, rawChart.Metadata.Version))
				return
			}
			return
		}
	}
	return
}

func buildChartInfo(rawChart *chart.Chart) (*release.ChartDetailInfo, error) {
	chartDetailInfo := new(release.ChartDetailInfo)

	chartDetailInfo.ChartName = rawChart.Metadata.Name
	chartDetailInfo.ChartVersion = rawChart.Metadata.Version
	chartDetailInfo.ChartAppVersion = rawChart.Metadata.AppVersion
	chartDetailInfo.ChartEngine = "transwarp"
	chartDetailInfo.ChartDescription = rawChart.Metadata.Description

	if len(rawChart.Values) != 0 {
		defaultValueBytes, err := json.Marshal(rawChart.Values)
		if err != nil {
			klog.Errorf("failed to marshal raw chart values: %s", err.Error())
			return nil, err
		}
		chartDetailInfo.DefaultValue = string(defaultValueBytes)
	}

	chartDetailInfo.WalmVersion = common.WalmVersionV2
	for _, f := range rawChart.Files {
		if f.Name == fmt.Sprintf(transwarpjsonnet.TranswarpAppYamlPattern, rawChart.Metadata.Name, rawChart.Metadata.AppVersion) {
			chartDetailInfo.WalmVersion = common.WalmVersionV1
			appMetaInfo := &TranswarpAppInfo{}
			err := yaml.Unmarshal(f.Data, &appMetaInfo)
			if err != nil {
				klog.Errorf("failed to unmarshal app yaml : %s", err.Error())
				return nil, err
			}
			for _, dependency := range appMetaInfo.Dependencies {
				dependency := release.ChartDependencyInfo {
					ChartName: dependency.Name,
					MaxVersion: dependency.MaxVersion,
					MinVersion: dependency.MinVersion,
					DependencyOptional: dependency.DependencyOptional,
					Requires: dependency.Requires,
				}
				chartDetailInfo.DependencyCharts = append(chartDetailInfo.DependencyCharts, dependency)
			}
			chartDetailInfo.ChartPrettyParams = convertUserInputParams(appMetaInfo.UserInputParams)
		}
		if f.Name == transwarpjsonnet.TranswarpMetadataDir+transwarpjsonnet.TranswarpArchitectureFileName {
			chartDetailInfo.Architecture = string(f.Data[:])
		}
		if f.Name == transwarpjsonnet.TranswarpMetadataDir+transwarpjsonnet.TranswarpAdvantageFileName {
			chartDetailInfo.Advantage = string(f.Data[:])
		}
		if f.Name == transwarpjsonnet.TranswarpMetadataDir+transwarpjsonnet.TranswarpIconFileName {
			chartDetailInfo.Icon = string(f.Data[:])
		}
	}

	chartMetaInfo, err := getChartMetaInfo(rawChart)
	if err != nil {
		klog.Errorf("failed to get chart meta info : %s", err.Error())
		return nil, err
	}
	chartDetailInfo.MetaInfo = chartMetaInfo

	if chartDetailInfo.MetaInfo != nil {
		chartDetailInfo.MetaInfo.BuildDefaultValue(chartDetailInfo.DefaultValue)
	}

	return chartDetailInfo, nil
}

func getChartIndexFile(repoURL, username, password string) (*repo.IndexFile, error) {
	repoIndex := &repo.IndexFile{}
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/") + "/index.yaml"

	indexURL := parsedURL.String()

	resp, err := resty.R().Get(indexURL)
	if err != nil {
		klog.Errorf("failed to get index : %s", err.Error())
		return nil, err
	}

	if err := yaml.Unmarshal(resp.Body(), repoIndex); err != nil {
		return nil, err
	}
	return repoIndex, nil
}

func loadChartFromRepo(repoUrl, username, password, chartName, chartVersion, dest string) (string, error) {
	indexFile, err := getChartIndexFile(repoUrl, username, password)
	if err != nil {
		klog.Errorf("failed to get chart index file : %s", err.Error())
		return "", err
	}

	cv, err := indexFile.Get(chartName, chartVersion)
	if err != nil {
		return "", fmt.Errorf("chart %s-%s is not found: %s", chartName, chartVersion, err.Error())
	}
	if len(cv.URLs) == 0 {
		return "", fmt.Errorf("chart %s has no downloadable URLs", chartName)
	}
	chartUrl := cv.URLs[0]
	absoluteChartURL, err := repo.ResolveReferenceURL(repoUrl, chartUrl)
	if err != nil {
		return "", fmt.Errorf("failed to make absolute chart url: %v", err)
	}
	resp, err := resty.R().Get(absoluteChartURL)
	if err != nil {
		klog.Errorf("failed to get chart : %s", err.Error())
		return "", err
	}

	name := filepath.Base(absoluteChartURL)
	destfile := filepath.Join(dest, name)
	if err := ioutil.WriteFile(destfile, resp.Body(), 0644); err != nil {
		klog.Errorf("failed to write file : %s", err.Error())
		return "", err
	}
	return destfile, nil
}

func (helmImpl *Helm) downloadChartFromRepo(repoName, chartName, version string) (string, error) {
	if repoName == "" {
		repoName = "stable"
	}
	repo, ok := helmImpl.chartRepoMap[repoName]
	if !ok {
		return "", fmt.Errorf("can not find repo name: %s", repoName)
	}
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	filename, err := loadChartFromRepo(repo.URL, repo.Username, repo.Password, chartName, version, tmpDir)
	if err != nil {
		klog.Infof("DownloadTo err %v", err)
		return "", err
	}

	return filename, nil
}
