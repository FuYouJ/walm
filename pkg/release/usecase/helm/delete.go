package helm

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/release"
	"encoding/json"
	"k8s.io/klog"
	"strings"
	"time"
)

func (helm *Helm) DeleteReleaseWithRetry(namespace, releaseName string, deletePvcs bool, async bool, timeoutSec int64) error {
	retryTimes := 5
	for {
		err := helm.DeleteRelease(namespace, releaseName, deletePvcs, async, timeoutSec)
		if err != nil {
			if (strings.Contains(err.Error(), release.WaitReleaseTaskMsgPrefix) || strings.Contains(err.Error(), release.SocketException)) && retryTimes > 0 {
				klog.Warningf("retry to delete release %s/%s after 2 second", namespace, releaseName)
				retryTimes--
				time.Sleep(time.Second * 2)
				continue
			}
		}
		return err
	}
}

func (helm *Helm) DeleteRelease(namespace, releaseName string, deletePvcs bool, async bool, timeoutSec int64) error {
	if timeoutSec == 0 {
		timeoutSec = defaultTimeoutSec
	}

	oldReleaseTask, err := helm.validateReleaseTask(namespace, releaseName, false)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("release task %s/%s is not found", namespace, releaseName)
			return nil
		}
		klog.Errorf("failed to validate release task : %s", err.Error())
		return err
	}

	releaseTaskArgs := &DeleteReleaseTaskArgs{
		Namespace:   namespace,
		ReleaseName: releaseName,
		DeletePvcs:  deletePvcs,
	}

	err = helm.sendReleaseTask(namespace, releaseName, deleteReleaseTaskName, releaseTaskArgs, oldReleaseTask, timeoutSec, async)
	if err != nil {
		klog.Errorf("async=%t, failed to send %s of %s/%s: %s", async, deleteReleaseTaskName, namespace, releaseName, err.Error())
		return err
	}
	klog.Infof("succeed to call delete release %s/%s api", namespace, releaseName)
	return nil
}

func (helm *Helm) doDeleteRelease(namespace, releaseName string, deletePvcs bool) error {
	releaseCache, err := helm.releaseCache.GetReleaseCache(namespace, releaseName)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("release cache %s is not found in redis", releaseName)
			return nil
		}
		klog.Errorf("failed to get release cache %s : %s", releaseName, err.Error())
		return err
	}
	releaseInfo, err := helm.buildReleaseInfoV2(releaseCache)
	if err != nil {
		klog.Errorf("failed to build release info : %s", err.Error())
		return err
	}

	err = helm.helm.DeleteRelease(namespace, releaseName)
	if err != nil {
		klog.Errorf("failed to delete release %s/%s from helm : %s", namespace, releaseName, err.Error())
		return err
	}

	err = helm.releaseCache.DeleteReleaseCache(namespace, releaseName)
	if err != nil {
		klog.Errorf("failed to delete release cache of %s : %s", releaseName, err.Error())
		return err
	}

	if deletePvcs {
		err = helm.k8sOperator.DeleteStatefulSetPvcs(releaseInfo.Status.StatefulSets)
		if err != nil {
			klog.Errorf("failed to delete stateful set pvcs : %s", err.Error())
			return err
		}
	}

	klog.Infof("succeed to delete release %s/%s", namespace, releaseName)

	releaseInfoByte, err := json.Marshal(releaseInfo)
	if err != nil {
		return err
	}
	err = helm.releaseCache.CreateReleaseBackUp(namespace, releaseName, releaseInfoByte)
	if err != nil {
		klog.Errorf("failed to backup releaseInfo of release which to be deleted : %s", err.Error())
		return err
	}

	return nil
}
