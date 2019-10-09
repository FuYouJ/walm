package usecase

import (
	"encoding/json"
	"fmt"
	"k8s.io/klog"
	errorModel "WarpCloud/walm/pkg/models/error"
)

const (
	removeReleaseTaskName = "Remove-Release-Task"
)

type RemoveReleaseTaskArgs struct {
	Namespace   string
	Name        string
	ReleaseName string
	DeletePvcs  bool
}

func (projectImpl *Project) registerRemoveReleaseTask() error {
	return projectImpl.task.RegisterTask(removeReleaseTaskName, projectImpl.RemoveReleaseTask)
}

func (projectImpl *Project) RemoveReleaseTask(removeReleaseTaskArgsStr string) error {
	removeReleaseTaskArgs := &RemoveReleaseTaskArgs{}
	err := json.Unmarshal([]byte(removeReleaseTaskArgsStr), removeReleaseTaskArgs)
	if err != nil {
		klog.Errorf("remove release task arg is not valid : %s", err.Error())
		return err
	}
	return projectImpl.doRemoveRelease(removeReleaseTaskArgs.Namespace, removeReleaseTaskArgs.Name, removeReleaseTaskArgs.ReleaseName, removeReleaseTaskArgs.DeletePvcs)
}

func (projectImpl *Project) doRemoveRelease(namespace, name, releaseName string, deletePvcs bool) error {
	projectInfo, err := projectImpl.GetProjectInfo(namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("project %s/%s is not found", namespace, name)
			return nil
		} else {
			klog.Errorf("failed to get project info : %s", err.Error())
			return err
		}
	}

	releaseParams := buildReleaseRequest(projectInfo, releaseName)
	if releaseParams == nil {
		return fmt.Errorf("release %s is not found in project %s", releaseName, name)
	}
	if projectInfo != nil {
		affectReleaseRequest, err2 := projectImpl.autoUpdateReleaseDependencies(projectInfo, releaseParams, true)
		if err2 != nil {
			klog.Errorf("RuntimeDepParse install release %s error %v\n", releaseParams.Name, err)
			return err2
		}
		for _, affectReleaseParams := range affectReleaseRequest {
			klog.Infof("Update BecauseOf Dependency Modified: %v", *affectReleaseParams)
			err = projectImpl.releaseUseCase.InstallUpgradeReleaseWithRetry(namespace, affectReleaseParams, nil, false, 0, nil)
			if err != nil {
				klog.Errorf("RemoveReleaseInProject Other Affected Release install release %s error %v\n", releaseParams.Name, err)
				return err
			}
		}
	}

	err = projectImpl.releaseUseCase.DeleteReleaseWithRetry(namespace, releaseName, deletePvcs, false, 0)
	if err != nil {
		klog.Errorf("failed to remove release %s/%s in project : %s", releaseName, name, err.Error())
		return err
	}
	return nil
}
