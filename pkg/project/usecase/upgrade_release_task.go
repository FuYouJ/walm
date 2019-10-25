package usecase

import (
	"encoding/json"
	"k8s.io/klog"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/release"
)

const (
	upgradeReleaseTaskName = "Upgrade-Release-Task"
)

type UpgradeReleaseTaskArgs struct {
	Namespace     string
	ProjectName   string
	ReleaseParams *release.ReleaseRequestV2
}

func (projectImpl *Project) registerUpgradeReleaseTask() error {
	return projectImpl.task.RegisterTask(upgradeReleaseTaskName, projectImpl.UpgradeReleaseTask)
}

func (projectImpl *Project) UpgradeReleaseTask(upgradeReleaseTaskArgsStr string) error {
	upgradeReleaseTaskArgs := &UpgradeReleaseTaskArgs{}
	err := json.Unmarshal([]byte(upgradeReleaseTaskArgsStr), upgradeReleaseTaskArgs)
	if err != nil {
		klog.Errorf("upgrade release task arg is not valid : %s", err.Error())
		return err
	}
	return projectImpl.upgradeRelease(upgradeReleaseTaskArgs.Namespace, upgradeReleaseTaskArgs.ProjectName, upgradeReleaseTaskArgs.ReleaseParams)
}

func (projectImpl *Project) upgradeRelease(namespace, projectName string, releaseParams *release.ReleaseRequestV2) (err error) {
	projectExists := true
	projectInfo, err := projectImpl.GetProjectInfo(namespace, projectName)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			projectExists = false
		} else {
			klog.Errorf("failed to get project info : %s", err.Error())
			return err
		}
	}

	setPrjLabelToReleaseParams(projectExists, projectInfo, releaseParams, projectName)

	err = projectImpl.releaseUseCase.InstallUpgradeReleaseWithRetry(namespace, releaseParams, nil, false, 0)
	if err != nil {
		klog.Errorf("failed to upgrade release %s in project %s/%s : %s", releaseParams.Name, namespace, projectName, err.Error())
		return
	}
	return
}
