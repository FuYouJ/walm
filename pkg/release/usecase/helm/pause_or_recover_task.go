package helm

import (
	"WarpCloud/walm/pkg/models/release"
	"encoding/json"
	"k8s.io/klog"
)

const (
	pauseOrRecoverReleaseTaskName = "Pause-Or-Recover-Release-Task"
)

type PauseOrRecoverReleaseTaskArgs struct {
	OldReleaseInfo *release.ReleaseInfoV2
	Paused         bool
}

func (helm *Helm) registerPauseOrRecoverReleaseTask() error {
	return helm.task.RegisterTask(pauseOrRecoverReleaseTaskName, helm.pauseOrRecoverReleaseTask)
}

func (helm *Helm) pauseOrRecoverReleaseTask(releaseTaskArgsStr string) error {
	releaseTaskArgs := &PauseOrRecoverReleaseTaskArgs{}
	err := json.Unmarshal([]byte(releaseTaskArgsStr), releaseTaskArgs)
	if err != nil {
		klog.Errorf("%s args is not valid : %s", pauseOrRecoverReleaseTaskName, err.Error())
		return err
	}
	err = helm.doPauseOrRecoverRelease(releaseTaskArgs.OldReleaseInfo, releaseTaskArgs.Paused)
	if err != nil {
		klog.Errorf("failed to %s release %s/%s : %s", buildActionMsg(releaseTaskArgs.Paused),
			releaseTaskArgs.OldReleaseInfo.Namespace, releaseTaskArgs.OldReleaseInfo.Name, err.Error())
		return err
	}
	return nil
}

func (helm *Helm) doPauseOrRecoverRelease(oldReleaseInfo *release.ReleaseInfoV2, paused bool) (error) {
	releaseCache, err := helm.helm.PauseOrRecoverRelease(paused, oldReleaseInfo)
	if err != nil {
		klog.Errorf("failed to %s release %s/%s : %s", buildActionMsg(paused), oldReleaseInfo.Namespace, oldReleaseInfo.Name, err.Error())
		return err
	}
	err = helm.releaseCache.CreateOrUpdateReleaseCache(releaseCache)
	if err != nil {
		klog.Errorf("failed to create of update release cache of %s/%s : %s", oldReleaseInfo.Namespace, oldReleaseInfo.Name, err.Error())
		return err
	}
	klog.Infof("succeed to %s release %s/%s", buildActionMsg(paused), oldReleaseInfo.Namespace, oldReleaseInfo.Name)

	return nil
}

func buildActionMsg(paused bool) string {
	if paused {
		return "pause"
	} else {
		return "recover"
	}
}

func (helm *Helm) PauseReleaseWithoutChart(namespace, releaseName string) error {
	releaseInfo, err := helm.GetRelease(namespace, releaseName)
	if err != nil {
		return err
	}

	replicas := int32(0)
	err = helm.k8sOperator.BackupAndUpdateReplicas(namespace, releaseName, releaseInfo.Status, replicas)
	if err != nil {
		klog.Errorf("failed to backup and update replicas of %s/%s : %s", namespace, releaseName, err.Error())
		return err
	}
	return nil
}

func (helm *Helm) RecoverReleaseWithoutChart(namespace, releaseName string) error {
	releaseInfo, err := helm.GetRelease(namespace, releaseName)
	if err != nil {
		return err
	}
	err = helm.k8sOperator.RecoverReplicas(namespace, releaseName, releaseInfo.Status)
	if err != nil {
		return err
	}
	return nil
}