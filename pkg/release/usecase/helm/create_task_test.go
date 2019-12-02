package helm

import (
	helmMocks "WarpCloud/walm/pkg/helm/mocks"
	k8sMocks "WarpCloud/walm/pkg/k8s/mocks"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release/mocks"
	taskMocks "WarpCloud/walm/pkg/task/mocks"
	redisExMocks "WarpCloud/walm/pkg/redis/mocks"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestHelm_createReleaseTask(t *testing.T) {
	var mockReleaseCache *mocks.Cache
	var mockHelm *helmMocks.Helm
	var mockK8sOperator *k8sMocks.Operator
	var mockK8sCache *k8sMocks.Cache
	var mockTask *taskMocks.Task
	var mockReleaseManager *Helm

	var mockTaskState *taskMocks.TaskState
	var mockRedisEx *redisExMocks.RedisEx

	refreshMocks := func() {
		mockReleaseCache = &mocks.Cache{}
		mockHelm = &helmMocks.Helm{}
		mockK8sOperator = &k8sMocks.Operator{}
		mockK8sCache = &k8sMocks.Cache{}
		mockTask = &taskMocks.Task{}

		mockTaskState = &taskMocks.TaskState{}

		mockTask.On("RegisterTask", mock.Anything, mock.Anything).Return(nil)

		mockRedisEx = &redisExMocks.RedisEx{}
		mockRedisEx.On("Init", mock.Anything).Return(nil)
		var err error
		mockReleaseManager, err = NewHelm(mockReleaseCache, mockHelm, mockK8sCache, mockK8sOperator, mockTask, mockRedisEx)
		assert.IsType(t, err, nil)
	}

	tests := []struct {
		initMock           func()
		releaseTaskArgsStr string
		err                error
	}{
		{
			initMock: func() {
				refreshMocks()
			},
			releaseTaskArgsStr: "notvalid",
			err : &json.SyntaxError{},
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errors.New(""))
			},
			releaseTaskArgsStr: "{\"ReleaseRequest\":{}}",
			err: errors.New("failed"),
		},
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(nil, errorModel.NotFoundError{})
				mockHelm.On("InstallOrCreateReleaseWithStrict", mock.Anything, mock.Anything, mock.Anything, mock.Anything, false, mock.Anything, mock.Anything).Return(&release.ReleaseCache{}, nil)
				mockReleaseCache.On("CreateOrUpdateReleaseCache", mock.Anything).Return(nil)
			},
			releaseTaskArgsStr: "{\"ReleaseRequest\":{}}",
			err: nil,
		},
	}

	for _, test := range tests {
		test.initMock()
		err := mockReleaseManager.createReleaseTask(test.releaseTaskArgsStr)
		assert.IsType(t, test.err, err)

		mockReleaseCache.AssertExpectations(t)
		mockHelm.AssertExpectations(t)
		mockK8sOperator.AssertExpectations(t)
		mockK8sCache.AssertExpectations(t)
		mockTask.AssertExpectations(t)

		mockTaskState.AssertExpectations(t)
	}

}
