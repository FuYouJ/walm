package helm

import (
	"WarpCloud/walm/pkg/models/common"
	"WarpCloud/walm/pkg/models/release"
	"encoding/json"
	"k8s.io/klog"
)

const (
	createReleaseTaskName = "Create-Release-Task"
)

type CreateReleaseTaskArgs struct {
	Namespace      string
	ReleaseRequest *release.ReleaseRequestV2
	ChartFiles     []*common.BufferedFile
	Paused         *bool
}

func (helm *Helm) registerCreateReleaseTask() error{
	return helm.task.RegisterTask(createReleaseTaskName, helm.createReleaseTask)
}

func (helm *Helm) createReleaseTask(releaseTaskArgsStr string) error {
	releaseTaskArgs := &CreateReleaseTaskArgs{}
	err := json.Unmarshal([]byte(releaseTaskArgsStr), releaseTaskArgs)
	if err != nil {
		klog.Errorf("%s args is not valid : %s", createReleaseTaskName, err.Error())
		return err
	}
	_, err = helm.doInstallUpgradeRelease(releaseTaskArgs.Namespace,
		releaseTaskArgs.ReleaseRequest, releaseTaskArgs.ChartFiles, false, releaseTaskArgs.Paused)
	if err != nil {
		klog.Errorf("failed to install or update release %s/%s : %s", releaseTaskArgs.Namespace, releaseTaskArgs.ReleaseRequest.Name, err.Error())
		return err
	}
	return nil
}
