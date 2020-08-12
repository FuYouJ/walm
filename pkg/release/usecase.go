package release

import (
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/common"
)

const (
	WaitReleaseTaskMsgPrefix = "please wait for the last release task"
	SocketException = "connection reset by peer"
)

type UseCase interface {
	GetRelease(namespace, name string) (releaseV2 *release.ReleaseInfoV2, err error)
	GetBackUpRelease(namespace string, name string) (*release.ReleaseInfoV2 , error)
	GetReleaseEvents(namespace, name string) (*k8s.EventList , error)
	ListReleases(namespace, filter string) ([]*release.ReleaseInfoV2, error)
	ListBackUpReleases(namespace string) ([]*release.ReleaseInfoV2, error)
	ListReleasesByLabels(namespace string, labelSelectorStr string) ([]*release.ReleaseInfoV2, error)
	DryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) ([]map[string]interface{}, error)
	DryRunUpdateRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) (*release.ReleaseDryRunUpdateInfo, error)
	ComputeResourcesByDryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) (*release.ReleaseResources, error)
	ComputeResourcesByGetRelease(namespace string, name string) (*release.ReleaseResources, error)
	DeleteReleaseWithRetry(namespace, releaseName string, deletePvcs bool, async bool, timeoutSec int64) error
	DeleteRelease(namespace, releaseName string, deletePvcs bool, async bool, timeoutSec int64) error
	// paused :
	// 1. nil: maintain pause state
	// 2. true: make release paused
	// 3. false: make release recovered
	InstallUpgradeReleaseWithRetry(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile, async bool, timeoutSec int64) error
	InstallUpgradeRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile, async bool, timeoutSec int64, fullUpdate bool, updateConfigMap bool) error
	ReloadRelease(namespace, name string) error
	RestartRelease(namespace, releaseName string) error
	RestartReleaseIsomate(namespace, releaseName, isomateName string) error
	PauseOrRecoverRelease(namespace, releaseName string, async bool, timeoutSec int64, paused bool) error

	UpdateReleaseIngress(namespace, name, ingressName string, requestBody *k8s.IngressRequestBody) error
	UpdateReleaseConfigMap(namespace, name, configMapName string, requestBody *k8s.ConfigMapRequestBody) error
	PauseReleaseWithoutChart(namespace string, name string) error
	RecoverReleaseWithoutChart(namespace string, name string) error
}
