package usecase

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/project"
	"WarpCloud/walm/pkg/util"
	"encoding/json"
	"k8s.io/klog"
)

const (
	addReleaseTaskName = "Add-Release-Task"
)

type AddReleaseTaskArgs struct {
	Namespace     string
	Name          string
	ProjectParams *project.ProjectParams
}

func (projectImpl *Project) registerAddReleaseTask() error {
	return projectImpl.task.RegisterTask(addReleaseTaskName, projectImpl.AddReleaseTask)
}

func (projectImpl *Project) AddReleaseTask(addReleaseTaskArgsStr string) error {
	addReleaseTaskArgs := &AddReleaseTaskArgs{}
	err := json.Unmarshal([]byte(addReleaseTaskArgsStr), addReleaseTaskArgs)
	if err != nil {
		klog.Errorf("add release task arg is not valid : %s", err.Error())
		return err
	}
	return projectImpl.doAddRelease(addReleaseTaskArgs.Namespace, addReleaseTaskArgs.Name, addReleaseTaskArgs.ProjectParams)
}

func (projectImpl *Project) doAddRelease(namespace, name string, projectParams *project.ProjectParams) error {
	projectInfo, err := projectImpl.GetProjectInfo(namespace, name)
	projectExists := true
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			projectExists = false
		} else {
			klog.Errorf("failed to get project info : %s", err.Error())
			return err
		}
	}

	for _, releaseParams := range projectParams.Releases {
		if releaseParams.ReleaseLabels == nil {
			releaseParams.ReleaseLabels = map[string]string{}
		}
		releaseParams.ReleaseLabels[project.ProjectNameLabelKey] = name
		releaseParams.ConfigValues = util.MergeValues(releaseParams.ConfigValues, projectParams.CommonValues, false)
	}
	releaseList, err := projectImpl.autoCreateReleaseDependencies(projectParams)
	if err != nil {
		klog.Errorf("failed to parse project charts dependency relation  : %s", err.Error())
		return err
	}

	for _, releaseParams := range releaseList {
		if projectExists {
			affectReleaseRequest, err2 := projectImpl.autoUpdateReleaseDependencies(projectInfo, releaseParams, false)
			if err2 != nil {
				klog.Errorf("RuntimeDepParse install release %s error %v\n", releaseParams.Name, err)
				return err2
			}
			err = projectImpl.releaseUseCase.InstallUpgradeReleaseWithRetry(namespace, releaseParams, nil, false, 0, nil)
			if err != nil {
				klog.Errorf("AddReleaseInProject install release %s error %v\n", releaseParams.Name, err)
				return err
			}
			for _, affectReleaseParams := range affectReleaseRequest {
				klog.Infof("Update BecauseOf Dependency Modified: %v", *affectReleaseParams)
				err = projectImpl.releaseUseCase.InstallUpgradeReleaseWithRetry(namespace, affectReleaseParams, nil, false, 0, nil)
				if err != nil {
					klog.Errorf("AddReleaseInProject Other Affected Release install release %s error %v\n", releaseParams.Name, err)
					return err
				}
			}
		} else {
			err = projectImpl.releaseUseCase.InstallUpgradeReleaseWithRetry(namespace, releaseParams, nil, false, 0, nil)
			if err != nil {
				klog.Errorf("AddReleaseInProject install release %s error %v\n", releaseParams.Name, err)
				return err
			}
		}
	}

	return nil
}
