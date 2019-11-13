package usecase

import (
	"WarpCloud/walm/pkg/models/project"
	"WarpCloud/walm/pkg/util"
	"encoding/json"
	"k8s.io/klog"
	errorModel "WarpCloud/walm/pkg/models/error"
)

const (
	createProjectTaskName = "Create-Project-Task"
)

type CreateProjectTaskArgs struct {
	Namespace     string
	Name          string
	ProjectParams *project.ProjectParams
}

func (projectImpl *Project) registerCreateProjectTask() error {
	return projectImpl.task.RegisterTask(createProjectTaskName, projectImpl.CreateProjectTask)
}

func (projectImpl *Project) CreateProjectTask(createProjectTaskArgsStr string) error {
	createProjectTaskArgs := &CreateProjectTaskArgs{}
	err := json.Unmarshal([]byte(createProjectTaskArgsStr), createProjectTaskArgs)
	if err != nil {
		klog.Errorf("create project task arg is not valid : %s", err.Error())
		return err
	}
	err = projectImpl.doCreateProject(createProjectTaskArgs.Namespace, createProjectTaskArgs.Name, createProjectTaskArgs.ProjectParams)
	if err != nil {
		klog.Errorf("failed to create project %s/%s : %s", createProjectTaskArgs.Namespace, createProjectTaskArgs.Name, err.Error())
		return err
	}
	return nil
}

func (projectImpl *Project) doCreateProject(namespace string, name string, projectParams *project.ProjectParams) error {
	projectExists := true
	projectInfo, err := projectImpl.GetProjectInfo(namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			projectExists = false
		} else {
			klog.Errorf("failed to get project info : %s", err.Error())
			return err
		}
	}

	rawValsBase := map[string]interface{}{}
	rawValsBase = util.MergeValues(rawValsBase, projectParams.CommonValues, false)

	for _, releaseParams := range projectParams.Releases {
		releaseParams.ConfigValues = util.MergeValues(releaseParams.ConfigValues, rawValsBase, false)
		setPrjLabelToReleaseParams(projectExists, projectInfo, releaseParams, name)
	}

	_, err = projectImpl.autoCreateReleaseDependencies(projectParams, namespace, true)
	if err != nil {
		klog.Errorf("failed to parse project charts dependency relation  : %s", err.Error())
		return err
	}

	return nil
}
