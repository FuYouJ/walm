package helm

import (
	"walm/pkg/release"
	"github.com/sirupsen/logrus"
	"walm/pkg/util"
	"walm/pkg/util/transwarpjsonnet"
	"k8s.io/helm/pkg/walm"
	"k8s.io/helm/pkg/walm/plugins"
	walmerr "walm/pkg/util/error"
	"k8s.io/helm/pkg/chart"
	"k8s.io/helm/pkg/chart/loader"
	"fmt"
	"k8s.io/helm/pkg/helm"
	hapirelease "k8s.io/helm/pkg/hapi/release"
	"walm/pkg/task"
	"time"
	"strings"
	"walm/pkg/release/manager/helm/cache"
)

func (hc *HelmClient) InstallUpgradeReleaseWithRetry(namespace string, releaseRequest *release.ReleaseRequestV2, isSystem bool, chartFiles []*loader.BufferedFile, async bool, timeoutSec int64) error {
	retryTimes := 5
	for {
		err := hc.InstallUpgradeRelease(namespace, releaseRequest, isSystem, chartFiles, async, timeoutSec)
		if err != nil {
			if strings.Contains(err.Error(), "please wait for the release latest task") && retryTimes > 0 {
				logrus.Warnf("retry to install or upgrade release %s/%s after 2 second", namespace, releaseRequest.Name)
				retryTimes --
				time.Sleep(time.Second * 2)
				continue
			}
		}
		return err
	}
}

func (hc *HelmClient) InstallUpgradeRelease(namespace string, releaseRequest *release.ReleaseRequestV2, isSystem bool, chartFiles []*loader.BufferedFile, async bool, timeoutSec int64) error {
	err := validateParams(releaseRequest, chartFiles)
	if err != nil {
		return err
	}

	if timeoutSec == 0 {
		timeoutSec = defaultTimeoutSec
	}

	oldReleaseTask, err := hc.validateReleaseTask(namespace, releaseRequest.Name, true)
	if err != nil {
		return err
	}

	releaseTaskArgs := &CreateReleaseTaskArgs{
		Namespace:      namespace,
		ReleaseRequest: releaseRequest,
		IsSystem:       isSystem,
		ChartFiles:     chartFiles,
	}
	taskSig, err := SendReleaseTask(releaseTaskArgs)
	if err != nil {
		logrus.Errorf("failed to send %s : %s", releaseTaskArgs.GetTaskName(), err.Error())
		return err
	}
	taskSig.TimeoutSec = timeoutSec

	releaseTask := &cache.ReleaseTask{
		Namespace:            namespace,
		Name:                 releaseRequest.Name,
		LatestReleaseTaskSig: taskSig,
	}

	err = hc.helmCache.CreateOrUpdateReleaseTask(releaseTask)
	if err != nil {
		logrus.Errorf("failed to set release task of %s/%s to redis: %s", namespace, releaseRequest.Name, err.Error())
		return err
	}

	if oldReleaseTask != nil && oldReleaseTask.LatestReleaseTaskSig != nil {
		err = task.GetDefaultTaskManager().PurgeTaskState(oldReleaseTask.LatestReleaseTaskSig.GetTaskSignature())
		if err != nil {
			logrus.Warnf("failed to purge task state : %s", err.Error())
		}
	}

	if !async {
		asyncResult := taskSig.GetAsyncResult()
		_, err = asyncResult.GetWithTimeout(time.Duration(timeoutSec)*time.Second, defaultSleepTimeSecond)
		if err != nil {
			logrus.Errorf("failed to create or update release  %s/%s: %s", namespace, releaseRequest.Name, err.Error())
			return err
		}
	}
	logrus.Infof("succeed to call create or update release %s/%s api", namespace, releaseRequest.Name)
	return nil
}

func validateParams(releaseRequest *release.ReleaseRequestV2, chartFiles []*loader.BufferedFile) error {
	if releaseRequest.Name == "" {
		return fmt.Errorf("release name can not be empty")
	}

	if releaseRequest.ChartName == "" && len(chartFiles) == 0 {
		return fmt.Errorf("chart name or chart should be supported")
	}

	return nil
}

func (hc *HelmClient) doInstallUpgradeRelease(namespace string, releaseRequest *release.ReleaseRequestV2, isSystem bool, chartFiles []*loader.BufferedFile) error {
	update := true
	releaseCache, err := hc.helmCache.GetReleaseCache(namespace, releaseRequest.Name)
	if err != nil {
		if walmerr.IsNotFoundError(err) {
			update = false
		} else {
			logrus.Errorf("failed to get release cache of %s/%s : %s", namespace, releaseRequest.Name, err.Error())
			return err
		}
	}

	preProcessRequest(releaseRequest)

	var rawChart *chart.Chart
	var chartErr error
	if chartFiles != nil {
		rawChart, chartErr = loader.LoadFiles(chartFiles)
	} else {
		rawChart, chartErr = hc.LoadChart(releaseRequest.RepoName, releaseRequest.ChartName, releaseRequest.ChartVersion)
	}
	if chartErr != nil {
		logrus.Errorf("failed to load chart : %s", chartErr.Error())
		return chartErr
	}

	chartInfo, err := BuildChartInfo(rawChart)
	if err != nil {
		logrus.Errorf("failed to build chart info : %s", err.Error())
		return err
	}
	// support meta pretty parameters
	configValues := releaseRequest.ConfigValues
	if releaseRequest.MetaInfoParams != nil {
		metaInfoConfigs, err := releaseRequest.MetaInfoParams.ToConfigValues(chartInfo.MetaInfo)
		if err != nil {
			logrus.Errorf("failed to get meta info parameters : %s", err.Error())
			return err
		}
		util.MergeValues(configValues, metaInfoConfigs)
	}

	dependencies := releaseRequest.Dependencies
	releaseLabels := releaseRequest.ReleaseLabels
	walmPlugins := releaseRequest.Plugins
	if update {
		// reuse config values, dependencies, release labels, walm plugins
		configValues, dependencies, releaseLabels, walmPlugins, err = hc.reuseReleaseRequest(releaseCache, releaseRequest)
		if err != nil {
			logrus.Errorf("failed to reuse release request : %s", err.Error())
			return err
		}
	}

	if chartInfo.MetaInfo != nil {
		walmPlugins, err = mergeWalmPlugins(walmPlugins, chartInfo.MetaInfo.Plugins)
		if err != nil {
			logrus.Errorf("failed to merge chart default plugins : %s", err.Error())
			return err
		}
	}

	// get all the dependency releases' output configs from ReleaseConfig
	dependencyConfigs, err := hc.getDependencyOutputConfigs(namespace, dependencies, chartInfo.MetaInfo)
	if err != nil {
		logrus.Errorf("failed to get all the dependency releases' output configs : %s", err.Error())
		return err
	}

	err = transwarpjsonnet.ProcessJsonnetChart(releaseRequest.RepoName, rawChart, namespace, releaseRequest.Name, configValues,
		dependencyConfigs, dependencies, releaseLabels)
	if err != nil {
		logrus.Errorf("failed to ProcessJsonnetChart : %s", err.Error())
		return err
	}

	// add default plugin
	walmPlugins = append(walmPlugins, &walm.WalmPlugin{
		Name: plugins.ValidateReleaseConfigPluginName,
	})

	valueOverride := map[string]interface{}{}
	util.MergeValues(valueOverride, configValues)
	util.MergeValues(valueOverride, dependencyConfigs)
	valueOverride[walm.WalmPluginConfigKey] = walmPlugins

	currentHelmClient, err := hc.getCurrentHelmClient(namespace)
	if err != nil {
		logrus.Errorf("failed to get helm client : %s", err.Error())
		return err
	}

	releaseInfo, err := hc.doInstallUpgradeReleaseFromChart(currentHelmClient, namespace, releaseRequest, rawChart, valueOverride, update)
	if err != nil {
		logrus.Errorf("failed to create or update release from chart : %s", err.Error())
		return err
	}

	err = hc.helmCache.CreateOrUpdateReleaseCache(releaseInfo)
	if err != nil {
		logrus.Errorf("failed to create of update release cache of %s/%s : %s", namespace, releaseRequest.Name, err.Error())
		return err
	}

	logrus.Infof("succeed to create or update release %s/%s", namespace, releaseRequest.Name)

	return nil
}

func preProcessRequest(releaseRequest *release.ReleaseRequestV2) {
	if releaseRequest.ConfigValues == nil {
		releaseRequest.ConfigValues = map[string]interface{}{}
	}
	if releaseRequest.Dependencies == nil {
		releaseRequest.Dependencies = map[string]string{}
	}
	if releaseRequest.ReleaseLabels == nil {
		releaseRequest.ReleaseLabels = map[string]string{}
	}
}

func (hc *HelmClient) reuseReleaseRequest(releaseCache *release.ReleaseCache, releaseRequest *release.ReleaseRequestV2) (
	configValues map[string]interface{}, dependencies map[string]string, releaseLabels map[string]string, walmPlugins []*walm.WalmPlugin, err error) {
	releaseInfo, err := hc.buildReleaseInfoV2(releaseCache)
	if err != nil {
		logrus.Errorf("failed to build release info : %s", err.Error())
		return nil, nil, nil, nil, err
	}

	configValues = map[string]interface{}{}
	util.MergeValues(configValues,	releaseInfo.ConfigValues)
	util.MergeValues(configValues, releaseRequest.ConfigValues)

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

	walmPlugins, err = mergeWalmPlugins(releaseRequest.Plugins, releaseInfo.Plugins)
	return
}

func mergeWalmPlugins(plugins, defaultPlugins []*walm.WalmPlugin) (mergedPlugins []*walm.WalmPlugin, err error) {
	walmPluginsMap := map[string]*walm.WalmPlugin{}
	for _, plugin := range plugins {
		if _, ok := walmPluginsMap[plugin.Name]; ok {
			return nil, fmt.Errorf("more than one plugin %s is not allowed", plugin.Name)
		} else {
			walmPluginsMap[plugin.Name] = plugin
		}
	}
	for _, plugin := range defaultPlugins {
		if _, ok := walmPluginsMap[plugin.Name]; !ok {
			walmPluginsMap[plugin.Name] = plugin
		}
	}
	for _, plugin := range walmPluginsMap {
		mergedPlugins = append(mergedPlugins, plugin)
	}
	return
}

func (hc *HelmClient) doInstallUpgradeReleaseFromChart(currentHelmClient *helm.Client, namespace string,
	releaseRequest *release.ReleaseRequestV2, rawChart *chart.Chart, valueOverride map[string]interface{},
	update bool) (releaseInfo *hapirelease.Release, err error) {
	if update {
		releaseInfo, err = currentHelmClient.UpdateReleaseFromChart(
			releaseRequest.Name,
			rawChart,
			helm.UpdateValueOverrides(valueOverride),
			helm.UpgradeDryRun(hc.dryRun),
		)
		if err != nil {
			logrus.Errorf("failed to upgrade release %s/%s from chart : %s", namespace, releaseRequest.Name, err.Error())
			return nil, err
		}
	} else {
		releaseInfo, err = currentHelmClient.InstallReleaseFromChart(
			rawChart,
			namespace,
			helm.ValueOverrides(valueOverride),
			helm.ReleaseName(releaseRequest.Name),
			helm.InstallDryRun(hc.dryRun),
		)
		if err != nil {
			logrus.Errorf("failed to install release %s/%s from chart : %s", namespace, releaseRequest.Name, err.Error())
			opts := []helm.UninstallOption{
				helm.UninstallPurge(true),
			}
			_, err1 := currentHelmClient.UninstallRelease(
				releaseRequest.Name, opts...,
			)
			if err1 != nil {
				logrus.Errorf("failed to rollback to delete release %s/%s : %s", namespace, releaseRequest.Name, err1.Error())
			}
			return nil, err
		}
	}
	return
}