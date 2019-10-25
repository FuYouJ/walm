package helm

import (
	"k8s.io/klog"
)

func (helm *Helm) PauseOrRecoverRelease(namespace, releaseName string, async bool, timeoutSec int64, paused bool) error {
	releaseInfo, err := helm.GetRelease(namespace, releaseName)
	if err != nil {
		klog.Errorf("failed to get release %s/%s : %s", namespace, releaseName, err.Error())
		return err
	}

	if paused {
		if releaseInfo.Paused {
			klog.Warningf("release %s/%s has already been paused", namespace, releaseName)
			return nil
		}
	} else {
		if !releaseInfo.Paused {
			klog.Warningf("release %s/%s is not paused", namespace, releaseName)
			return nil
		}
	}

	if timeoutSec == 0 {
		timeoutSec = defaultTimeoutSec
	}

	oldReleaseTask, err := helm.validateReleaseTask(namespace, releaseName, false)
	if err != nil {
		klog.Errorf("failed to validate release task : %s", err.Error())
		return err
	}

	releaseTaskArgs := &PauseOrRecoverReleaseTaskArgs{
		OldReleaseInfo: releaseInfo,
		Paused: paused,
	}

	err = helm.sendReleaseTask(namespace, releaseName, pauseOrRecoverReleaseTaskName, releaseTaskArgs, oldReleaseTask, timeoutSec, async)
	if err != nil {
		klog.Errorf("async=%t, failed to send %s of %s/%s: %s", async, pauseOrRecoverReleaseTaskName, namespace, releaseName, err.Error())
		return err
	}
	klog.Infof("succeed to call %s release %s/%s api", buildActionMsg(paused), namespace, releaseName)
	return nil
}
