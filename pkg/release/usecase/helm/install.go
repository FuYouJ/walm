package helm

import (
	"WarpCloud/walm/pkg/models/common"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/release"
	"fmt"
	"k8s.io/klog"
	"strings"
	"time"

	releasei "WarpCloud/walm/pkg/release"
)

const (
	defaultTimeoutSec int64 = 60 * 5
)

func (helm *Helm) InstallUpgradeReleaseWithRetry(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile, async bool, timeoutSec int64) error {
	retryTimes := 5
	for {
		err := helm.installUpgradeReleaseWithStrict(namespace, releaseRequest, chartFiles, async, timeoutSec, false)
		if err != nil {
			if strings.Contains(err.Error(), releasei.WaitReleaseTaskMsgPrefix) && retryTimes > 0 {
				klog.Warningf("retry to install or upgrade release %s/%s after 15 second", namespace, releaseRequest.Name)
				retryTimes--
				time.Sleep(time.Second * 15)
				continue
			}
		}
		return err
	}
}

func (helm *Helm) InstallUpgradeRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile, async bool, timeoutSec int64) error {
	return helm.installUpgradeReleaseWithStrict(namespace, releaseRequest, chartFiles, async, timeoutSec, true)
}

// strict: describe whether to allow that the dependency release does not exist. true: not allow, false: allow.
func (helm *Helm) installUpgradeReleaseWithStrict(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile, async bool, timeoutSec int64, strict bool) error {
	err := validateParams(releaseRequest, chartFiles)
	if err != nil {
		klog.Errorf("failed to validate params : %s", err.Error())
		return err
	}

	if timeoutSec == 0 {
		timeoutSec = defaultTimeoutSec
	}

	oldReleaseTask, err := helm.validateReleaseTask(namespace, releaseRequest.Name, true)
	if err != nil {
		return err
	}

	releaseTaskArgs := &CreateReleaseTaskArgs{
		Namespace:      namespace,
		ReleaseRequest: releaseRequest,
		ChartFiles:     chartFiles,
		Strict:         strict,
	}

	err = helm.sendReleaseTask(namespace, releaseRequest.Name, createReleaseTaskName, releaseTaskArgs, oldReleaseTask, timeoutSec, async)
	if err != nil {
		klog.Errorf("async=%t, failed to send %s of %s/%s: %s", async, createReleaseTaskName, namespace, releaseRequest.Name, err.Error())
		return err
	}
	klog.Infof("succeed to call create or update release %s/%s api", namespace, releaseRequest.Name)
	return nil
}

func validateParams(releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) error {
	if releaseRequest.Name == "" {
		return fmt.Errorf("release name can not be empty")
	}

	if releaseRequest.ChartName == "" && releaseRequest.ChartImage == "" && len(chartFiles) == 0 {
		return fmt.Errorf("at lease one of chart name or chart image or chart files should be supported")
	}

	if releaseRequest.IsomateConfig != nil {
		if len(releaseRequest.IsomateConfig.Isomates) == 0 {
			return fmt.Errorf("at lease one isomate should be supported")
		}
		isomateNames := map[string]bool{}
		for _, isomate := range releaseRequest.IsomateConfig.Isomates {
			if isomate.Name == "" {
				return fmt.Errorf("isomate name can not be empty")
			}
			if _, ok := isomateNames[isomate.Name]; ok {
				return fmt.Errorf("duplicate isomate name %s is not allowed", isomate.Name)
			} else {
				isomateNames[isomate.Name] = true
			}
		}
		if releaseRequest.IsomateConfig.DefaultIsomateName != "" {
			if _, ok := isomateNames[releaseRequest.IsomateConfig.DefaultIsomateName]; !ok {
				return fmt.Errorf("default isomate name %s does not exist", releaseRequest.IsomateConfig.DefaultIsomateName)
			}
		}
	}

	return nil
}

func (helm *Helm) doInstallUpgradeRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile, dryRun bool) (*release.ReleaseCache, error) {
	return helm.doInstallUpgradeReleaseWithStrict(namespace, releaseRequest, chartFiles, dryRun, true)
}

func (helm *Helm) doInstallUpgradeReleaseWithStrict(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile, dryRun bool, strict bool) (*release.ReleaseCache, error) {
	update := true
	oldReleaseCache, err := helm.releaseCache.GetReleaseCache(namespace, releaseRequest.Name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			update = false
		} else {
			klog.Errorf("failed to get release cache of %s/%s : %s", namespace, releaseRequest.Name, err.Error())
			return nil, err
		}
	}

	var oldReleaseInfo *release.ReleaseInfoV2
	if oldReleaseCache != nil {
		oldReleaseInfo, err = helm.buildReleaseInfoV2(oldReleaseCache)
		if err != nil {
			klog.Errorf("failed to build release info of %s/%s: %s", namespace, releaseRequest.Name, err.Error())
			return nil, err
		}
	}

	preProcessRequest(releaseRequest)

	releaseCache, err := helm.helm.InstallOrCreateReleaseWithStrict(namespace, releaseRequest, chartFiles, dryRun, update, oldReleaseInfo, strict)
	if err != nil {
		klog.Errorf("failed to install or update release %s/%s : %s", namespace, releaseRequest.Name, err.Error())
		return nil, err
	}
	if !dryRun {
		err = helm.releaseCache.CreateOrUpdateReleaseCache(releaseCache)
		if err != nil {
			klog.Errorf("failed to create of update release cache of %s/%s : %s", namespace, releaseRequest.Name, err.Error())
			return nil, err
		}
		klog.Infof("succeed to create or update release %s/%s", namespace, releaseRequest.Name)
	} else {
		klog.Infof("succeed to dry run create or update release %s/%s", namespace, releaseRequest.Name)
	}

	return releaseCache, nil
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
