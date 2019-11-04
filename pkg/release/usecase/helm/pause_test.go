package helm

import (
	helmMocks "WarpCloud/walm/pkg/helm/mocks"
	k8sMocks "WarpCloud/walm/pkg/k8s/mocks"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/task"
	"WarpCloud/walm/pkg/release/mocks"
	taskMocks "WarpCloud/walm/pkg/task/mocks"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"WarpCloud/walm/pkg/helm/impl/plugins"
)

func TestHelm_PauseRelease(t *testing.T) {
	var mockReleaseCache *mocks.Cache
	var mockHelm *helmMocks.Helm
	var mockK8sOperator *k8sMocks.Operator
	var mockK8sCache *k8sMocks.Cache
	var mockTask *taskMocks.Task
	var mockReleaseManager *Helm

	var mockTaskState *taskMocks.TaskState

	refreshMocks := func() {
		mockReleaseCache = &mocks.Cache{}
		mockHelm = &helmMocks.Helm{}
		mockK8sOperator = &k8sMocks.Operator{}
		mockK8sCache = &k8sMocks.Cache{}
		mockTask = &taskMocks.Task{}

		mockTaskState = &taskMocks.TaskState{}

		mockTask.On("RegisterTask", mock.Anything, mock.Anything).Return(nil)

		var err error
		mockReleaseManager, err = NewHelm(mockReleaseCache, nil, mockHelm, mockK8sCache, mockK8sOperator, mockTask)
		assert.IsType(t, err, nil)
	}

	tests := []struct {
		initMock func()
		paused   bool
		err      error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockReleaseCache.On("GetReleaseTask", mock.Anything, mock.Anything).Return(nil, errors.New("failed"))
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
				mockReleaseCache.On("GetReleaseCache", mock.Anything, mock.Anything).Return(&release.ReleaseCache{
					ReleaseSpec: release.ReleaseSpec{
						Namespace: "test-ns",
						Name:      "test-name",
					},
					ComputedValues: map[string]interface{}{
						plugins.WalmPluginConfigKey: []interface{}{
							map[string]interface{}{
								"name":    "test-plugin",
								"args":    "test-args",
								"version": "test-version",
								"disable": false,
							},
							map[string]interface{}{
								"name":    plugins.PauseReleasePluginName,
								"args":    "",
								"version": "",
								"disable": false,
							},
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
			paused: true,
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
						ChartName: "test-name",
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

				mockTask.On("SendTask", mock.Anything, mock.Anything, mock.Anything).Return(&task.TaskSig{}, nil)
				mockReleaseCache.On("CreateOrUpdateReleaseTask", mock.Anything).Return(nil)
				mockTask.On("TouchTask", mock.Anything, mock.Anything).Return(nil)
				mockTask.On("PurgeTaskState", mock.Anything).Return(nil)
			},
			paused: true,
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
			paused: false,
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
						ChartName: "test-name",
					},
					ComputedValues: map[string]interface{}{
						plugins.WalmPluginConfigKey: []interface{}{
							map[string]interface{}{
								"name":    "test-plugin",
								"args":    "test-args",
								"version": "test-version",
								"disable": false,
							},
							map[string]interface{}{
								"name":    plugins.PauseReleasePluginName,
								"args":    "",
								"version": "",
								"disable": false,
							},
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

				mockTask.On("SendTask", mock.Anything, mock.Anything, mock.Anything).Return(&task.TaskSig{}, nil)
				mockReleaseCache.On("CreateOrUpdateReleaseTask", mock.Anything).Return(nil)
				mockTask.On("TouchTask", mock.Anything, mock.Anything).Return(nil)
				mockTask.On("PurgeTaskState", mock.Anything).Return(nil)
			},
			paused: false,
			err: nil,
		},
	}

	for _, test := range tests {
		test.initMock()
		err := mockReleaseManager.PauseOrRecoverRelease("test-ns", "test-name", false, 0, test.paused)
		assert.IsType(t, test.err, err)

		mockReleaseCache.AssertExpectations(t)
		mockHelm.AssertExpectations(t)
		mockK8sOperator.AssertExpectations(t)
		mockK8sCache.AssertExpectations(t)
		mockTask.AssertExpectations(t)

		mockTaskState.AssertExpectations(t)
	}

}
