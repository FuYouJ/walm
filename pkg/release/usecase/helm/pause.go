package helm

import "k8s.io/klog"

func (helm *Helm) PauseRelease(namespace, releaseName string, async bool, timeoutSec int64) error {
	releaseInfo, err := helm.GetRelease(namespace, releaseName)
	if err != nil {
		klog.Errorf("failed to get release %s/%s : %s", namespace, releaseName, err.Error())
		return err
	}

	if releaseInfo.Paused {
		klog.Warningf("release %s/%s has already been paused", namespace, releaseName)
		return nil
	}

	releaseRequest := releaseInfo.BuildReleaseRequestV2()
	paused := true
	err = helm.InstallUpgradeRelease(namespace, releaseRequest, nil, async, timeoutSec, &paused)
	if err != nil {
		klog.Errorf("failed to upgrade release %s/%s : %s", namespace, releaseName, err.Error())
		return err
	}
	return nil

}
