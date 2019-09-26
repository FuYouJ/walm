package project

import (
	"WarpCloud/walm/pkg/models/common"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/task"
)

const (
	ProjectNameLabelKey = "Project-Name"
)

type ProjectParams struct {
	CommonValues map[string]interface{}      `json:"commonValues" description:"common values added to the chart"`
	Releases     []*release.ReleaseRequestV2 `json:"releases" description:"list of release of the project"`
}

type ProjectInfo struct {
	Name        string                   `json:"name" description:"project name"`
	Namespace   string                   `json:"namespace" description:"project namespace"`
	Releases    []*release.ReleaseInfoV2 `json:"releases" description:"list of release of the project"`
	Ready       bool                     `json:"ready" description:"whether all the project releases are ready"`
	Message     string                   `json:"message" description:"why project is not ready"`
	WalmVersion common.WalmVersion       `json:"walmVersion" description:"chart walm version: v1, v2"`
}

type ProjectInfoList struct {
	Num   int            `json:"num" description:"project number"`
	Items []*ProjectInfo `json:"items" description:"project info list"`
}

type ProjectTask struct {
	Name                string             `json:"name" description:"project name"`
	Namespace           string             `json:"namespace" description:"project namespace"`
	WalmVersion         common.WalmVersion `json:"walmVersion" description:"chart walm version: v1, v2"`
	LatestTaskSignature *task.TaskSig      `json:"latestTaskSignature" description:"latest task signature"`
	// compatible
	LatestTaskTimeoutSec int64 `json:"latestTaskTimeoutSec" description:"latest task timeout sec"`
}

func (projectTask *ProjectTask) CompatiblePreviousProjectTask() {
	if projectTask.LatestTaskSignature != nil {
		if projectTask.LatestTaskSignature.TimeoutSec == 0 {
			projectTask.LatestTaskSignature.TimeoutSec = projectTask.LatestTaskTimeoutSec
		}
	}
}
