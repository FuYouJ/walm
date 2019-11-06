package helm

import (
	redisExMocks "WarpCloud/walm/pkg/redis/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"

	helmMocks "WarpCloud/walm/pkg/helm/mocks"
	k8sMocks "WarpCloud/walm/pkg/k8s/mocks"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/task"
	"WarpCloud/walm/pkg/release/mocks"
	taskMocks "WarpCloud/walm/pkg/task/mocks"
)

func TestHelm_UpdateReleaseConfigMap(t *testing.T) {
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
		initMock func()
		err      error
	}{
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
				mockK8sOperator.On("UpdateConfigMap", mock.Anything, mock.Anything).Return(nil)
			},
			err: errorModel.NotFoundError{},
		},
	}

	for _, test := range tests {
		test.initMock()
		err := mockReleaseManager.UpdateReleaseConfigMap("test-ns", "test-name", "ingress", &k8s.ConfigMapRequestBody{
			Data: map[string]string{
				"test": "test Data",
			},
		})
		assert.IsType(t, test.err, err)
	}
}
