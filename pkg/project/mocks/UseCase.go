// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	project "WarpCloud/walm/pkg/models/project"

	release "WarpCloud/walm/pkg/models/release"
)

// UseCase is an autogenerated mock type for the UseCase type
type UseCase struct {
	mock.Mock
}

// AddReleasesInProject provides a mock function with given fields: namespace, projectName, projectParams, async, timeoutSec
func (_m *UseCase) AddReleasesInProject(namespace string, projectName string, projectParams *project.ProjectParams, async bool, timeoutSec int64) ([]string, error) {
	ret := _m.Called(namespace, projectName, projectParams, async, timeoutSec)

	var r0 []string
	if rf, ok := ret.Get(0).(func(string, string, *project.ProjectParams, bool, int64) []string); ok {
		r0 = rf(namespace, projectName, projectParams, async, timeoutSec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, *project.ProjectParams, bool, int64) error); ok {
		r1 = rf(namespace, projectName, projectParams, async, timeoutSec)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ComputeResourcesByDryRunProject provides a mock function with given fields: namespace, projectName, projectParams
func (_m *UseCase) ComputeResourcesByDryRunProject(namespace string, projectName string, projectParams *project.ProjectParams) ([]release.ReleaseResourcesInfo, error) {
	ret := _m.Called(namespace, projectName, projectParams)

	var r0 []release.ReleaseResourcesInfo
	if rf, ok := ret.Get(0).(func(string, string, *project.ProjectParams) []release.ReleaseResourcesInfo); ok {
		r0 = rf(namespace, projectName, projectParams)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]release.ReleaseResourcesInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, *project.ProjectParams) error); ok {
		r1 = rf(namespace, projectName, projectParams)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ComputeResourcesByGetProject provides a mock function with given fields: namespace, projectName
func (_m *UseCase) ComputeResourcesByGetProject(namespace string, projectName string) ([]release.ReleaseResourcesInfo, error) {
	ret := _m.Called(namespace, projectName)

	var r0 []release.ReleaseResourcesInfo
	if rf, ok := ret.Get(0).(func(string, string) []release.ReleaseResourcesInfo); ok {
		r0 = rf(namespace, projectName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]release.ReleaseResourcesInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, projectName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateProject provides a mock function with given fields: namespace, _a1, projectParams, async, timeoutSec
func (_m *UseCase) CreateProject(namespace string, _a1 string, projectParams *project.ProjectParams, async bool, timeoutSec int64) error {
	ret := _m.Called(namespace, _a1, projectParams, async, timeoutSec)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, *project.ProjectParams, bool, int64) error); ok {
		r0 = rf(namespace, _a1, projectParams, async, timeoutSec)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteProject provides a mock function with given fields: namespace, _a1, async, timeoutSec, deletePvcs
func (_m *UseCase) DeleteProject(namespace string, _a1 string, async bool, timeoutSec int64, deletePvcs bool) error {
	ret := _m.Called(namespace, _a1, async, timeoutSec, deletePvcs)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, bool, int64, bool) error); ok {
		r0 = rf(namespace, _a1, async, timeoutSec, deletePvcs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DryRunProject provides a mock function with given fields: namespace, projectName, projectParams
func (_m *UseCase) DryRunProject(namespace string, projectName string, projectParams *project.ProjectParams) ([]map[string]interface{}, error) {
	ret := _m.Called(namespace, projectName, projectParams)

	var r0 []map[string]interface{}
	if rf, ok := ret.Get(0).(func(string, string, *project.ProjectParams) []map[string]interface{}); ok {
		r0 = rf(namespace, projectName, projectParams)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]map[string]interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, *project.ProjectParams) error); ok {
		r1 = rf(namespace, projectName, projectParams)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetProjectInfo provides a mock function with given fields: namespace, projectName
func (_m *UseCase) GetProjectInfo(namespace string, projectName string) (*project.ProjectInfo, error) {
	ret := _m.Called(namespace, projectName)

	var r0 *project.ProjectInfo
	if rf, ok := ret.Get(0).(func(string, string) *project.ProjectInfo); ok {
		r0 = rf(namespace, projectName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*project.ProjectInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, projectName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListProjects provides a mock function with given fields: namespace
func (_m *UseCase) ListProjects(namespace string) (*project.ProjectInfoList, error) {
	ret := _m.Called(namespace)

	var r0 *project.ProjectInfoList
	if rf, ok := ret.Get(0).(func(string) *project.ProjectInfoList); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*project.ProjectInfoList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveReleaseInProject provides a mock function with given fields: namespace, projectName, releaseName, async, timeoutSec, deletePvcs
func (_m *UseCase) RemoveReleaseInProject(namespace string, projectName string, releaseName string, async bool, timeoutSec int64, deletePvcs bool) error {
	ret := _m.Called(namespace, projectName, releaseName, async, timeoutSec, deletePvcs)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, bool, int64, bool) error); ok {
		r0 = rf(namespace, projectName, releaseName, async, timeoutSec, deletePvcs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpgradeReleaseInProject provides a mock function with given fields: namespace, projectName, releaseParams, async, timeoutSec
func (_m *UseCase) UpgradeReleaseInProject(namespace string, projectName string, releaseParams *release.ReleaseRequestV2, async bool, timeoutSec int64) error {
	ret := _m.Called(namespace, projectName, releaseParams, async, timeoutSec)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, *release.ReleaseRequestV2, bool, int64) error); ok {
		r0 = rf(namespace, projectName, releaseParams, async, timeoutSec)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
