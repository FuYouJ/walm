package helm

import (
	helmMocks "WarpCloud/walm/pkg/helm/mocks"
	k8sMocks "WarpCloud/walm/pkg/k8s/mocks"
	"WarpCloud/walm/pkg/models/common"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/task"
	redisExMocks "WarpCloud/walm/pkg/redis/mocks"
	"WarpCloud/walm/pkg/release/mocks"
	taskMocks "WarpCloud/walm/pkg/task/mocks"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestHelm_GetRelease(t *testing.T) {

	var mockReleaseCache *mocks.Cache
	var mockHelm *helmMocks.Helm
	var mockK8sOperator *k8sMocks.Operator
	var mockK8sCache *k8sMocks.Cache
	var mockTask *taskMocks.Task
	var mockReleaseManager *Helm
	var mockRedisEx *redisExMocks.RedisEx

	var mockTaskState *taskMocks.TaskState

	refreshMocks := func() {
		mockReleaseCache = &mocks.Cache{}
		mockHelm = &helmMocks.Helm{}
		mockK8sOperator = &k8sMocks.Operator{}
		mockK8sCache = &k8sMocks.Cache{}
		mockTask = &taskMocks.Task{}
		mockRedisEx = &redisExMocks.RedisEx{}

		mockTaskState = &taskMocks.TaskState{}

		mockTask.On("RegisterTask", mock.Anything, mock.Anything).Return(nil)
		mockRedisEx.On("Init", mock.Anything).Return(nil)

		var err error
		mockReleaseManager, err = NewHelm(mockReleaseCache, mockHelm, mockK8sCache, mockK8sOperator, mockTask, mockRedisEx)
		assert.IsType(t, err, nil)
	}

	testResourceSet := k8s.NewResourceSet()
	testResourceSet.StatefulSets = append(testResourceSet.StatefulSets, &k8s.StatefulSet{
		Meta: k8s.NewMeta("", "", "", k8s.NewState("Pending", "", "")),
	})

	tests := []struct {
		initMock    func()
		releaseInfo *release.ReleaseInfoV2
		err         error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(nil, errors.New("failed"))
			},
			releaseInfo: nil,
			err:         errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errors.New("failed"))
			},
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					RealName: "test-name",
				},
			},
			err: errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errorModel.NotFoundError{})
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(nil, errors.New("failed"))
			},
			releaseInfo: nil,
			err:         errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errorModel.NotFoundError{})
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(nil, errorModel.NotFoundError{})
			},
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					RealName: "test-name",
				},
			},
			err: nil,
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errorModel.NotFoundError{})
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(mockTaskState, nil)
				mockTaskState.On("IsFinished").Return(true)
				mockTaskState.On("IsSuccess").Return(true)
			},
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					RealName: "test-name",
				},
			},
			err: nil,
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errorModel.NotFoundError{})
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(mockTaskState, nil)
				mockTaskState.On("IsFinished").Return(true)
				mockTaskState.On("IsSuccess").Return(false)
				mockTaskState.On("GetErrorMsg").Return("test-err")
			},
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					RealName: "test-name",
					Message:  "the release latest task test-name-test-uuid failed : test-err",
				},
				MsgCode: release.ReleaseFailed,
			},
			err: nil,
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errorModel.NotFoundError{})
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(mockTaskState, nil)
				mockTaskState.On("IsFinished").Return(false)
			},
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					RealName: "test-name",
					Message:  "please wait for the release latest task test-name-test-uuid finished",
				},
				MsgCode: release.ReleasePending,
			},
			err: nil,
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(&release.ReleaseCache{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
				}, nil)
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(mockTaskState, nil)
				mockTaskState.On("IsFinished").Return(true)
				mockTaskState.On("IsSuccess").Return(true)
				mockK8sCache.On("GetResourceSet", ([]release.ReleaseResourceMeta)(nil)).Return(k8s.NewResourceSet(), nil)
				mockK8sCache.On("GetResource", k8s.ReleaseConfigKind, "test-ns", "test-name").Return(&k8s.ReleaseConfig{}, nil)
			},
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					RealName: "test-name",
					Ready:    true,
					Status:   k8s.NewResourceSet(),
				},
				Plugins:            []*k8s.ReleasePlugin{},
				ReleaseWarmVersion: common.WalmVersionV2,
			},
			err: nil,
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(&release.ReleaseTask{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}, nil)
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(&release.ReleaseCache{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
				}, nil)
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(mockTaskState, nil)
				mockTaskState.On("IsFinished").Return(true)
				mockTaskState.On("IsSuccess").Return(true)
				mockK8sCache.On("GetResourceSet", ([]release.ReleaseResourceMeta)(nil)).Return(testResourceSet, nil)
				mockK8sCache.On("GetResource", k8s.ReleaseConfigKind, "test-ns", "test-name").Return(&k8s.ReleaseConfig{}, nil)
			},
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					RealName: "test-name",
					Ready:    false,
					Message:  " / is in state Pending",
					Status:   testResourceSet,
				},
				Plugins:            []*k8s.ReleasePlugin{},
				ReleaseWarmVersion: common.WalmVersionV2,
				MsgCode:            release.ReleaseNotReady,
			},
			err: nil,
		},
	}

	for _, test := range tests {
		test.initMock()
		releaseInfo, err := mockReleaseManager.GetRelease("test-ns", "test-name")
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.releaseInfo, releaseInfo)

		mockReleaseCache.AssertExpectations(t)
		mockHelm.AssertExpectations(t)
		mockK8sOperator.AssertExpectations(t)
		mockK8sCache.AssertExpectations(t)
		mockTask.AssertExpectations(t)

		mockTaskState.AssertExpectations(t)
	}
}

func TestHelm_buildReleaseInfo(t *testing.T) {

	var mockReleaseCache *mocks.Cache
	var mockHelm *helmMocks.Helm
	var mockK8sOperator *k8sMocks.Operator
	var mockK8sCache *k8sMocks.Cache
	var mockTask *taskMocks.Task
	var mockReleaseManager *Helm
	var mockRedisEx *redisExMocks.RedisEx

	var mockTaskState *taskMocks.TaskState

	refreshMocks := func() {
		mockReleaseCache = &mocks.Cache{}
		mockHelm = &helmMocks.Helm{}
		mockK8sOperator = &k8sMocks.Operator{}
		mockK8sCache = &k8sMocks.Cache{}
		mockTask = &taskMocks.Task{}
		mockRedisEx = &redisExMocks.RedisEx{}

		mockTaskState = &taskMocks.TaskState{}

		mockTask.On("RegisterTask", mock.Anything, mock.Anything).Return(nil)
		mockRedisEx.On("Init", mock.Anything).Return(nil)

		var err error
		mockReleaseManager, err = NewHelm(mockReleaseCache, mockHelm, mockK8sCache, mockK8sOperator, mockTask, mockRedisEx)
		assert.IsType(t, err, nil)
	}

	tests := []struct {
		initMock     func()
		releaseCache *release.ReleaseCache
		releaseInfo  *release.ReleaseInfo
		err          error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResourceSet", ([]release.ReleaseResourceMeta)(nil)).Return(nil, errors.New("failed"))

			},
			releaseCache: &release.ReleaseCache{},
			releaseInfo:  &release.ReleaseInfo{},
			err:          errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResourceSet", ([]release.ReleaseResourceMeta)(nil)).Return(&k8s.ResourceSet{
					Deployments: []*k8s.Deployment{
						{
							Meta: k8s.Meta{
								Namespace: "test-ns",
								Name:      "test-name",
								Kind:      "Deployment",
								State:     k8s.State{Status: "Pending"},
							},
							ExpectedReplicas: 2,
						},
					},
				}, nil)
			},
			releaseCache: &release.ReleaseCache{},
			releaseInfo: &release.ReleaseInfo{
				Status: &k8s.ResourceSet{
					Deployments: []*k8s.Deployment{
						{
							Meta: k8s.Meta{
								Namespace: "test-ns",
								Name:      "test-name",
								Kind:      "Deployment",
								State:     k8s.State{Status: "Pending"},
							},
							ExpectedReplicas: 2,
						},
					},
				},
				Ready:   false,
				Message: "Deployment test-ns/test-name is in state Pending",
			},
			err: nil,
		},
	}

	for _, test := range tests {
		test.initMock()
		releaseInfo, err := mockReleaseManager.buildReleaseInfo(test.releaseCache)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.releaseInfo, releaseInfo)

		mockReleaseCache.AssertExpectations(t)
		mockHelm.AssertExpectations(t)
		mockK8sOperator.AssertExpectations(t)
		mockK8sCache.AssertExpectations(t)
		mockTask.AssertExpectations(t)

		mockTaskState.AssertExpectations(t)
	}
}

func TestHelm_ListReleases(t *testing.T) {
	var mockReleaseCache *mocks.Cache
	var mockHelm *helmMocks.Helm
	var mockK8sOperator *k8sMocks.Operator
	var mockK8sCache *k8sMocks.Cache
	var mockTask *taskMocks.Task
	var mockReleaseManager *Helm
	var mockRedisEx *redisExMocks.RedisEx

	var mockTaskState *taskMocks.TaskState

	refreshMocks := func() {
		mockReleaseCache = &mocks.Cache{}
		mockHelm = &helmMocks.Helm{}
		mockK8sOperator = &k8sMocks.Operator{}
		mockK8sCache = &k8sMocks.Cache{}
		mockTask = &taskMocks.Task{}

		mockTaskState = &taskMocks.TaskState{}
		mockRedisEx = &redisExMocks.RedisEx{}

		mockTask.On("RegisterTask", mock.Anything, mock.Anything).Return(nil)
		mockRedisEx.On("Init", mock.Anything).Return(nil)

		var err error
		mockReleaseManager, err = NewHelm(mockReleaseCache, mockHelm, mockK8sCache, mockK8sOperator, mockTask, mockRedisEx)
		assert.IsType(t, err, nil)
	}

	tests := []struct {
		initMock     func()
		releaseInfos []*release.ReleaseInfoV2
		err          error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTasks", mock.Anything, mock.Anything).Return(nil, errors.New("failed"))
			},
			releaseInfos: nil,
			err:          errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTasks", mock.Anything, mock.Anything).Return([]*release.ReleaseTask{}, nil)
				mockReleaseCache.On("GetReleaseCaches", mock.Anything, mock.Anything).Return(nil, errors.New("failed"))
			},
			releaseInfos: nil,
			err:          errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTasks", mock.Anything, mock.Anything).Return([]*release.ReleaseTask{{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}}, nil)
				mockReleaseCache.On("GetReleaseCaches", mock.Anything, mock.Anything).Return([]*release.ReleaseCache{
					{
						ReleaseSpec: release.ReleaseSpec{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}, nil)
				mockK8sCache.On("GetResourceSet", ([]release.ReleaseResourceMeta)(nil)).Return(nil, errors.New("failed"))
			},
			releaseInfos: []*release.ReleaseInfoV2{},
			err:          errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTasks", mock.Anything, mock.Anything).Return([]*release.ReleaseTask{{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}}, nil)
				mockReleaseCache.On("GetReleaseCaches", mock.Anything, mock.Anything).Return([]*release.ReleaseCache{
					{
						ReleaseSpec: release.ReleaseSpec{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}, nil)
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(mockTaskState, nil)
				mockTaskState.On("IsFinished").Return(true)
				mockTaskState.On("IsSuccess").Return(true)
				mockK8sCache.On("GetResourceSet", ([]release.ReleaseResourceMeta)(nil)).Return(k8s.NewResourceSet(), nil)
				mockK8sCache.On("GetResource", k8s.ReleaseConfigKind, "test-ns", "test-name").Return(&k8s.ReleaseConfig{}, nil)
			},
			releaseInfos: []*release.ReleaseInfoV2{
				{
					ReleaseInfo: release.ReleaseInfo{
						ReleaseSpec: release.ReleaseSpec{
							Namespace: "test-ns",
							Name:      "test-name",
						},
						RealName: "test-name",
						Ready:    true,
						Status:   k8s.NewResourceSet(),
					},
					Plugins:            []*k8s.ReleasePlugin{},
					ReleaseWarmVersion: common.WalmVersionV2,
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		test.initMock()
		releaseInfos, err := mockReleaseManager.ListReleases("test-ns", "")
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.releaseInfos, releaseInfos)

		mockReleaseCache.AssertExpectations(t)
		mockHelm.AssertExpectations(t)
		mockK8sOperator.AssertExpectations(t)
		mockK8sCache.AssertExpectations(t)
		mockTask.AssertExpectations(t)

		mockTaskState.AssertExpectations(t)
	}

}

func TestHelm_ListReleasesByLabels(t *testing.T) {
	var mockReleaseCache *mocks.Cache
	var mockHelm *helmMocks.Helm
	var mockK8sOperator *k8sMocks.Operator
	var mockK8sCache *k8sMocks.Cache
	var mockTask *taskMocks.Task
	var mockReleaseManager *Helm
	var mockRedisEx *redisExMocks.RedisEx
	var mockTaskState *taskMocks.TaskState

	refreshMocks := func() {
		mockReleaseCache = &mocks.Cache{}
		mockHelm = &helmMocks.Helm{}
		mockK8sOperator = &k8sMocks.Operator{}
		mockK8sCache = &k8sMocks.Cache{}
		mockTask = &taskMocks.Task{}
		mockRedisEx = &redisExMocks.RedisEx{}
		mockTaskState = &taskMocks.TaskState{}

		mockTask.On("RegisterTask", mock.Anything, mock.Anything).Return(nil)
		mockRedisEx.On("Init", mock.Anything).Return(nil)
		var err error
		mockReleaseManager, err = NewHelm(mockReleaseCache, mockHelm, mockK8sCache, mockK8sOperator, mockTask, mockRedisEx)
		assert.IsType(t, err, nil)
	}

	tests := []struct {
		initMock     func()
		releaseInfos []*release.ReleaseInfoV2
		err          error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("ListReleaseConfigs", mock.Anything, mock.Anything).Return(nil, errors.New("failed"))
			},
			releaseInfos: nil,
			err:          errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("ListReleaseConfigs", mock.Anything, mock.Anything).Return([]*k8s.ReleaseConfig{}, nil)
			},
			releaseInfos: []*release.ReleaseInfoV2{},
			err:          nil,
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("ListReleaseConfigs", mock.Anything, mock.Anything).Return([]*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}, nil)
				mockReleaseCache.On("GetReleaseTasksByReleaseConfigs", []*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}).Return(nil, errors.New("failed"))
			},
			releaseInfos: nil,
			err:          errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("ListReleaseConfigs", mock.Anything, mock.Anything).Return([]*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}, nil)
				mockReleaseCache.On("GetReleaseTasksByReleaseConfigs", []*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}).Return([]*release.ReleaseTask{{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}}, nil)
				mockReleaseCache.On("GetReleaseCachesByReleaseConfigs", []*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}).Return(nil, errors.New("failed"))
			},
			releaseInfos: nil,
			err:          errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("ListReleaseConfigs", mock.Anything, mock.Anything).Return([]*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}, nil)
				mockReleaseCache.On("GetReleaseTasksByReleaseConfigs", []*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}).Return([]*release.ReleaseTask{{
					Namespace: "test-ns",
					Name:      "test-name",
					LatestReleaseTaskSig: &task.TaskSig{
						Name: "test-name",
						UUID: "test-uuid",
					},
				}}, nil)
				mockReleaseCache.On("GetReleaseCachesByReleaseConfigs", []*k8s.ReleaseConfig{
					{
						Meta: k8s.Meta{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}).Return([]*release.ReleaseCache{
					{
						ReleaseSpec: release.ReleaseSpec{
							Namespace: "test-ns",
							Name:      "test-name",
						},
					},
				}, nil)
				mockTask.On("GetTaskState", &task.TaskSig{
					Name: "test-name",
					UUID: "test-uuid",
				}).Return(mockTaskState, nil)
				mockTaskState.On("IsFinished").Return(true)
				mockTaskState.On("IsSuccess").Return(true)
				mockK8sCache.On("GetResourceSet", ([]release.ReleaseResourceMeta)(nil)).Return(k8s.NewResourceSet(), nil)
				mockK8sCache.On("GetResource", k8s.ReleaseConfigKind, "test-ns", "test-name").Return(&k8s.ReleaseConfig{}, nil)
			},
			releaseInfos: []*release.ReleaseInfoV2{
				{
					ReleaseInfo: release.ReleaseInfo{
						ReleaseSpec: release.ReleaseSpec{
							Namespace: "test-ns",
							Name:      "test-name",
						},
						RealName: "test-name",
						Ready:    true,
						Status:   k8s.NewResourceSet(),
					},
					Plugins:            []*k8s.ReleasePlugin{},
					ReleaseWarmVersion: common.WalmVersionV2,
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		test.initMock()
		releaseInfos, err := mockReleaseManager.ListReleasesByLabels("test-ns", "")
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.releaseInfos, releaseInfos)

		mockReleaseCache.AssertExpectations(t)
		mockHelm.AssertExpectations(t)
		mockK8sOperator.AssertExpectations(t)
		mockK8sCache.AssertExpectations(t)
		mockTask.AssertExpectations(t)

		mockTaskState.AssertExpectations(t)
	}

}

func Test_buildReleaseFailedMsgCode(t *testing.T) {
	tests := []struct {
		taskName       string
		releaseExisted bool
		msgCode        release.ReleaseMsgCode
	}{
		{
			taskName:       createReleaseTaskName,
			releaseExisted: true,
			msgCode:        release.ReleaseUpgradeFailed,
		},
		{
			taskName:       createReleaseTaskName,
			releaseExisted: false,
			msgCode:        release.ReleaseInstallFailed,
		},
		{
			taskName: deleteReleaseTaskName,
			msgCode:  release.ReleaseDeleteFailed,
		},
		{
			taskName: pauseOrRecoverReleaseTaskName,
			msgCode:  release.ReleasePauseOrRecoverFailed,
		},
		{
			taskName: "unknown",
			msgCode:  release.ReleaseFailed,
		},
	}

	for _, test := range tests {
		msgCode := buildReleaseFailedMsgCode(test.taskName, test.releaseExisted)
		assert.Equal(t, test.msgCode, msgCode)
	}
}

func Test_buildReleaseNotReadyMsgCode(t *testing.T) {
	tests := []struct {
		paused  bool
		msgCode release.ReleaseMsgCode
	}{
		{
			paused: false,
			msgCode: release.ReleaseNotReady,
		},
		{
			paused: true,
			msgCode: release.ReleasePaused,
		},
	}

	for _, test := range tests {
		msgCode := buildReleaseNotReadyMsgCode(test.paused)
		assert.Equal(t, test.msgCode, msgCode)
	}
}
