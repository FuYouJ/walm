package helm

import (
	"WarpCloud/walm/pkg/helm"
	"WarpCloud/walm/pkg/k8s"
	errorModel "WarpCloud/walm/pkg/models/error"
	releaseModel "WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release"
	"WarpCloud/walm/pkg/release/utils"
	"WarpCloud/walm/pkg/task"
	"fmt"
	"k8s.io/klog"
)

type Helm struct {
	releaseCache release.Cache
	helm         helm.Helm
	k8sCache     k8s.Cache
	k8sOperator  k8s.Operator
	task         task.Task
}

// reload dependencies config values, if changes, upgrade release
func (helm *Helm) ReloadRelease(namespace, name string) error {
	releaseInfo, err := helm.GetRelease(namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("release %s/%s is not found， ignore to reload release", namespace, name)
			return nil
		}
		klog.Errorf("failed to get release %s/%s : %s", namespace, name, err.Error())
		return err
	}

	chartInfo, err := helm.helm.GetChartDetailInfo(releaseInfo.RepoName, releaseInfo.ChartName, releaseInfo.ChartVersion)
	if err != nil {
		klog.Errorf("failed to get chart info : %s", err.Error())
		return err
	}

	oldDependenciesConfigValues := releaseInfo.DependenciesConfigValues
	newDependenciesConfigValues, err := helm.helm.GetDependencyOutputConfigs(namespace, releaseInfo.Dependencies, chartInfo.MetaInfo)
	if err != nil {
		klog.Errorf("failed to get dependencies output configs of %s/%s : %s", namespace, name, err.Error())
		return err
	}

	if utils.ConfigValuesDiff(oldDependenciesConfigValues, newDependenciesConfigValues) {
		releaseRequest := releaseInfo.BuildReleaseRequestV2()
		err = helm.InstallUpgradeRelease(namespace, releaseRequest, nil, false, 0, nil)
		if err != nil {
			klog.Errorf("failed to upgrade release v2 %s/%s : %s", namespace, name, err.Error())
			return err
		}
		klog.Infof("succeed to reload release %s/%s", namespace, name)
	} else {
		klog.Infof("ignore reloading release %s/%s : dependencies config value does not change", namespace, name)
	}

	return nil
}

func (helm *Helm) validateReleaseTask(namespace, name string, allowReleaseTaskNotExist bool) (releaseTask *releaseModel.ReleaseTask, err error) {
	releaseTask, err = helm.releaseCache.GetReleaseTask(namespace, name)
	if err != nil {
		if !errorModel.IsNotFoundError(err) {
			klog.Errorf("failed to get release task : %s", err.Error())
			return
		} else if !allowReleaseTaskNotExist {
			return
		} else {
			err = nil
		}
	} else {
		taskState, err := helm.task.GetTaskState(releaseTask.LatestReleaseTaskSig)
		if err != nil {
			if errorModel.IsNotFoundError(err) {
				err = nil
				return releaseTask, err
			} else {
				klog.Errorf("failed to get the last release task state : %s", err.Error())
				return releaseTask, err
			}
		}

		if !(taskState.IsFinished() || taskState.IsTimeout()) {
			err = fmt.Errorf(release.WaitReleaseTaskMsgPrefix+" %s-%s finished or timeout", releaseTask.LatestReleaseTaskSig.Name, releaseTask.LatestReleaseTaskSig.UUID)
			klog.Warning(err.Error())
			return releaseTask, err
		}
	}
	return
}

func NewHelm(releaseCache release.Cache, helm helm.Helm, k8sCache k8s.Cache, k8sOperator k8s.Operator, task task.Task) (*Helm, error) {
	h := &Helm{
		releaseCache: releaseCache,
		helm:         helm,
		k8sCache:     k8sCache,
		k8sOperator:  k8sOperator,
		task:         task,
	}
	err := h.registerCreateReleaseTask()
	if err != nil {
		return nil, err
	}
	err = h.registerDeleteReleaseTask()
	if err != nil {
		return nil, err
	}
	return h, nil
}
