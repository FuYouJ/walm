package usecase

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	"encoding/json"
	"k8s.io/klog"
)

const (
	deleteProjectTaskName = "Delete-Project-Task"
)

type DeleteProjectTaskArgs struct {
	Namespace  string
	Name       string
	DeletePvcs bool
}

func (projectImpl *Project) registerDeleteProjectTask() error {
	return projectImpl.task.RegisterTask(deleteProjectTaskName, projectImpl.DeleteProjectTask)
}

func (projectImpl *Project) DeleteProjectTask(deleteProjectTaskArgsStr string) error {
	deleteProjectTaskArgs := &DeleteProjectTaskArgs{}
	err := json.Unmarshal([]byte(deleteProjectTaskArgsStr), deleteProjectTaskArgs)
	if err != nil {
		klog.Errorf("delete project task arg is not valid : %s", err.Error())
		return err
	}
	err = projectImpl.doDeleteProject(deleteProjectTaskArgs.Namespace, deleteProjectTaskArgs.Name, deleteProjectTaskArgs.DeletePvcs)
	if err != nil {
		klog.Errorf("failed to delete project %s/%s : %s", deleteProjectTaskArgs.Namespace, deleteProjectTaskArgs.Name, err.Error())
		return err
	}
	return nil
}

func (projectImpl *Project) doDeleteProject(namespace, name string, deletePvcs bool) error {
	projectInfo, err := projectImpl.GetProjectInfo(namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("project %s/%s is not found", namespace, name)
			return nil
		}
		klog.Errorf("failed to get project info %s/%s: %s", namespace, name, err.Error())
		return err
	}

	for _, releaseInfo := range projectInfo.Releases {
		err = projectImpl.releaseUseCase.DeleteReleaseWithRetry(namespace, releaseInfo.Name, deletePvcs, false, 0)
		if err != nil {
			klog.Errorf("failed to delete release %s/%s : %s", namespace, releaseInfo.Name, err.Error())
			return err
		}
	}

	err = projectImpl.cache.DeleteProjectTask(namespace, name)
	if err != nil {
		klog.Warningf("failed to delete project task %s/%s : %s", namespace, name, err.Error())
	}

	return nil
}
