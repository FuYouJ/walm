package helm

import (
	walmHelm "WarpCloud/walm/pkg/helm"
	k8sutils "WarpCloud/walm/pkg/k8s/utils"
	"WarpCloud/walm/pkg/models/common"
	errorModel "WarpCloud/walm/pkg/models/error"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	releaseModel "WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/util/transwarpjsonnet"
	"errors"
	"fmt"
	"k8s.io/klog"
	"sync"
)

const (
	releasePausedConfigKey = "Transwarp_Application_Pause"
)

func (helm *Helm) GetRelease(namespace, name string) (releaseV2 *releaseModel.ReleaseInfoV2, err error) {
	releaseTask, err := helm.releaseCache.GetReleaseTask(namespace, name)
	if err != nil {
		return nil, err
	}

	return helm.buildReleaseInfoV2ByReleaseTask(releaseTask, nil)
}

func(helm *Helm) GetBackUpRelease(namespace string, name string) (releaseRequestV2 *releaseModel.ReleaseInfoV2 , err error) {
	releaseRequestV2, err = helm.releaseCache.GetReleaseBackUp(namespace, name)
	if err != nil {
		return nil, err
	}
	return
}

func (helm *Helm) buildReleaseInfoV2ByReleaseTask(releaseTask *releaseModel.ReleaseTask, releaseCache *releaseModel.ReleaseCache) (releaseV2 *releaseModel.ReleaseInfoV2, err error) {
	releaseV2 = &releaseModel.ReleaseInfoV2{
		ReleaseInfo: releaseModel.ReleaseInfo{
			ReleaseSpec: releaseModel.ReleaseSpec{
				Namespace: releaseTask.Namespace,
				Name:      releaseTask.Name,
			},
			RealName: releaseTask.Name,
		},
	}

	if releaseCache == nil {
		releaseCache, err = helm.releaseCache.GetReleaseCache(releaseTask.Namespace, releaseTask.Name)
		if err != nil {
			if errorModel.IsNotFoundError(err) {
				klog.Warningf("release cache %s/%s is not found in redis", releaseTask.Namespace, releaseTask.Name)
				err = nil
			} else {
				klog.Errorf("failed to get release cache of %s/%s : %s", releaseTask.Namespace, releaseTask.Name, err.Error())
				return
			}
		}
	}

	releaseExisted := false
	if releaseCache != nil {
		releaseExisted = true
		releaseV2, err = helm.buildReleaseInfoV2(releaseCache)
		if err != nil {
			klog.Errorf("failed to build v2 release info : %s", err.Error())
			return
		}
	}

	taskState, err := helm.task.GetTaskState(releaseTask.LatestReleaseTaskSig)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			err = nil
			return releaseV2, nil
		} else {
			klog.Errorf("failed to get task state : %s", err.Error())
			return nil, err
		}
	}

	if taskState.IsFinished() {
		if !taskState.IsSuccess() {
			releaseV2.MsgCode = buildReleaseFailedMsgCode(releaseTask.LatestReleaseTaskSig.Name, releaseExisted)
			releaseV2.Message = fmt.Sprintf("the release latest task %s-%s failed : %s", releaseTask.LatestReleaseTaskSig.Name, releaseTask.LatestReleaseTaskSig.UUID, taskState.GetErrorMsg())
		}
	} else {
		releaseV2.Message = fmt.Sprintf("please wait for the release latest task %s-%s finished", releaseTask.LatestReleaseTaskSig.Name, releaseTask.LatestReleaseTaskSig.UUID)
		releaseV2.MsgCode = releaseModel.ReleasePending
	}

	return
}

func buildReleaseFailedMsgCode(taskName string, releaseExisted bool) (msgCode releaseModel.ReleaseMsgCode) {
	switch taskName{
	case createReleaseTaskName:
		if releaseExisted {
			msgCode = releaseModel.ReleaseUpgradeFailed
		}else {
			msgCode = releaseModel.ReleaseInstallFailed
		}
	case deleteReleaseTaskName:
		msgCode = releaseModel.ReleaseDeleteFailed
	case pauseOrRecoverReleaseTaskName:
		msgCode = releaseModel.ReleasePauseOrRecoverFailed
	default:
		msgCode = releaseModel.ReleaseFailed
	}
	return
}

func convertHelmVersionToWalmVersion(helmVersion string) common.WalmVersion {
	if helmVersion == "v3" {
		return common.WalmVersionV2
	}
	if helmVersion == "v2" {
		return common.WalmVersionV1
	}
	return common.WalmVersionV2
}

func (helm *Helm) buildReleaseInfoV2(releaseCache *releaseModel.ReleaseCache) (*releaseModel.ReleaseInfoV2, error) {
	releaseV1, err := helm.buildReleaseInfo(releaseCache)
	if err != nil {
		klog.Errorf("failed to build release info: %s", err.Error())
		return nil, err
	}

	releaseV2 := &releaseModel.ReleaseInfoV2{
		ReleaseInfo:        *releaseV1,
		ReleaseWarmVersion: convertHelmVersionToWalmVersion(releaseCache.HelmVersion),
		ComputedValues:     releaseCache.ComputedValues,
	}

	if releaseV2.ReleaseWarmVersion == common.WalmVersionV2 {
		releaseConfigResource, err := helm.k8sCache.GetResource(k8sModel.ReleaseConfigKind, releaseCache.Namespace, releaseCache.Name)
		if err != nil {
			if errorModel.IsNotFoundError(err) {
				releaseV2.DependenciesConfigValues = map[string]interface{}{}
				releaseV2.OutputConfigValues = map[string]interface{}{}
				releaseV2.ReleaseLabels = map[string]string{}
			} else {
				klog.Errorf("failed to get release config : %s", err.Error())
				return nil, err
			}
		} else {
			releaseConfig := releaseConfigResource.(*k8sModel.ReleaseConfig)
			releaseV2.ConfigValues = releaseConfig.ConfigValues
			releaseV2.CreationTimestamp = releaseConfig.CreationTimestamp
			releaseV2.Dependencies = releaseConfig.Dependencies
			releaseV2.DependenciesConfigValues = releaseConfig.DependenciesConfigValues
			releaseV2.OutputConfigValues = releaseConfig.OutputConfig
			releaseV2.ReleaseLabels = releaseConfig.Labels
			releaseV2.RepoName = releaseConfig.Repo
			releaseV2.ChartImage = releaseConfig.ChartImage
			releaseV2.IsomateConfig = releaseConfig.IsomateConfig
			releaseV2.ChartWalmVersion = releaseConfig.ChartWalmVersion
		}

		releaseV2.MetaInfoValues = releaseCache.MetaInfoValues
		releaseV2.PrettyParams = releaseCache.PrettyParams
		releaseV2.Plugins, releaseV2.Paused, err = walmHelm.BuildReleasePluginsByConfigValues(releaseV2.ComputedValues)
	} else if releaseV2.ReleaseWarmVersion == common.WalmVersionV1 {
		releaseV2.DependenciesConfigValues = map[string]interface{}{}
		releaseV2.OutputConfigValues = map[string]interface{}{}
		releaseV2.ReleaseLabels = map[string]string{}
		releaseV2.Plugins = []*k8sModel.ReleasePlugin{}
		releaseV2.Paused = buildV1ReleasePauseInfo(releaseV2.ConfigValues)
		releaseV2.ChartWalmVersion = common.WalmVersionV1

		instResource, err := helm.k8sCache.GetResource(k8sModel.InstanceKind, releaseCache.Namespace, releaseCache.Name)
		if err != nil {
			if !errorModel.IsNotFoundError(err) {
				klog.Errorf("failed to get instance : %s", err.Error())
				return nil, err
			}
		} else {
			instance := instResource.(*k8sModel.ApplicationInstance)
			releaseV2.CreationTimestamp = instance.CreationTimestamp
			releaseV2.Dependencies = instance.Dependencies
			releaseV2.OutputConfigValues = k8sutils.ConvertDependencyMetaToOutputConfig(instance.DependencyMeta)
			releaseV2.ConfigValues[transwarpjsonnet.TranswarpInstallIDKey] = instance.InstanceId
		}
	}

	if releaseV2.Paused {
		releaseV2.Ready = false
		releaseV2.Message = "Release is paused now"
	}

	if !releaseV2.Ready {
		releaseV2.MsgCode = buildReleaseNotReadyMsgCode(releaseV2.Paused)
	}

	return releaseV2, nil
}

func buildReleaseNotReadyMsgCode(paused bool) (msgCode releaseModel.ReleaseMsgCode) {
	if paused {
		msgCode = releaseModel.ReleasePaused
	} else {
		msgCode = releaseModel.ReleaseNotReady
	}
	return
}

// for compatible
func buildV1ReleasePauseInfo(ConfigValues map[string]interface{}) bool {
	if pausedValue, ok := ConfigValues[releasePausedConfigKey]; ok {
		if paused, ok1 := pausedValue.(bool); ok1 && paused {
			return true
		}
	}
	return false
}

func (helm *Helm) buildReleaseInfo(releaseCache *releaseModel.ReleaseCache) (releaseInfo *releaseModel.ReleaseInfo, err error) {
	releaseInfo = &releaseModel.ReleaseInfo{}
	releaseInfo.ReleaseSpec = releaseCache.ReleaseSpec
	releaseInfo.RealName = releaseCache.Name

	releaseInfo.Status, err = helm.k8sCache.GetResourceSet(releaseCache.ReleaseResourceMetas)
	if err != nil {
		klog.Errorf(fmt.Sprintf("Failed to build the status of releaseInfo: %s", releaseInfo.Name))
		return
	}
	ready, notReadyResource := releaseInfo.Status.IsReady()
	if ready {
		releaseInfo.Ready = true
	} else {
		releaseInfo.Message = fmt.Sprintf("%s %s/%s is in state %s", notReadyResource.GetKind(), notReadyResource.GetNamespace(), notReadyResource.GetName(), notReadyResource.GetState().Status)
	}

	return
}

func (helm *Helm) GetReleaseEvents(namespace, name string) (*k8sModel.EventList , error) {

	key := namespace + "/" + name
	var eventList k8sModel.EventList
	err := helm.redisEx.GetFieldValue(key, &eventList)
	//res := value.(*k8sModel.EventList)

	if err != nil {
		klog.Errorf("failed to get release events: %s", err.Error())
		return nil, err
	}
	return &eventList, nil
}

func (helm *Helm) ListReleases(namespace, filter string) ([]*releaseModel.ReleaseInfoV2, error) {
	releaseTasks, err := helm.releaseCache.GetReleaseTasks(namespace, filter)
	if err != nil {
		klog.Errorf("failed to get release tasks with namespace=%s : %s", namespace, err.Error())
		return nil, err
	}

	releaseCaches, err := helm.releaseCache.GetReleaseCaches(namespace, filter)
	if err != nil {
		klog.Errorf("failed to get release caches with namespace=%s : %s", namespace, err.Error())
		return nil, err
	}

	return helm.doListReleases(releaseTasks, releaseCaches)
}

func (helm *Helm) ListReleasesByLabels(namespace string, labelSelectorStr string) ([]*releaseModel.ReleaseInfoV2, error) {
	releaseConfigs, err := helm.k8sCache.ListReleaseConfigs(namespace, labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to list release configs : %s", err.Error())
		return nil, err
	}

	return helm.listReleasesByReleaseConfigs(releaseConfigs)
}

func(helm *Helm) ListBackUpReleases(namespace string) ([]*releaseModel.ReleaseInfoV2, error) {

	releaseInfoV2List, err := helm.releaseCache.ListReleasesBackUp(namespace)
	if err != nil {
		klog.Errorf("failed to list backup releases : %s", err.Error())
		return nil, err
	}
	return releaseInfoV2List, nil
}

func (helm *Helm) listReleasesByReleaseConfigs(releaseConfigs []*k8sModel.ReleaseConfig) ([]*releaseModel.ReleaseInfoV2, error) {
	if len(releaseConfigs) == 0 {
		return []*releaseModel.ReleaseInfoV2{}, nil
	}
	releaseTasks, err := helm.releaseCache.GetReleaseTasksByReleaseConfigs(releaseConfigs)
	if err != nil {
		klog.Errorf("failed to get release tasks : %s", err.Error())
		return nil, err
	}

	releaseCaches, err := helm.releaseCache.GetReleaseCachesByReleaseConfigs(releaseConfigs)
	if err != nil {
		klog.Errorf("failed to get release caches : %s", err.Error())
		return nil, err
	}

	return helm.doListReleases(releaseTasks, releaseCaches)
}

func (helm *Helm) doListReleases(releaseTasks []*releaseModel.ReleaseTask, releaseCaches []*releaseModel.ReleaseCache) (releaseInfos []*releaseModel.ReleaseInfoV2, err error) {
	releaseCacheMap := map[string]*releaseModel.ReleaseCache{}
	for _, releaseCache := range releaseCaches {
		releaseCacheMap[releaseCache.Namespace+"/"+releaseCache.Name] = releaseCache
	}

	releaseInfos = []*releaseModel.ReleaseInfoV2{}
	//TODO 限制协程的数量
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	for _, releaseTask := range releaseTasks {
		wg.Add(1)
		go func(releaseTask *releaseModel.ReleaseTask, releaseCache *releaseModel.ReleaseCache) {
			defer wg.Done()
			info, err1 := helm.buildReleaseInfoV2ByReleaseTask(releaseTask, releaseCache)
			if err1 != nil {
				err = errors.New(fmt.Sprintf("failed to build release info: %s", err1.Error()))
				klog.Error(err.Error())
				return
			}
			mux.Lock()
			releaseInfos = append(releaseInfos, info)
			mux.Unlock()
		}(releaseTask, releaseCacheMap[releaseTask.Namespace+"/"+releaseTask.Name])
	}
	wg.Wait()
	if err != nil {
		return
	}
	return
}
