// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	modelsk8s "WarpCloud/walm/pkg/models/k8s"

	mock "github.com/stretchr/testify/mock"

	release "WarpCloud/walm/pkg/models/release"
)

// Operator is an autogenerated mock type for the Operator type
type Operator struct {
	mock.Mock
}

// AnnotateNode provides a mock function with given fields: name, annotationsToAdd, annotationsToRemove
func (_m *Operator) AnnotateNode(name string, annotationsToAdd map[string]string, annotationsToRemove []string) error {
	ret := _m.Called(name, annotationsToAdd, annotationsToRemove)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, map[string]string, []string) error); ok {
		r0 = rf(name, annotationsToAdd, annotationsToRemove)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BuildManifestObjects provides a mock function with given fields: namespace, manifest
func (_m *Operator) BuildManifestObjects(namespace string, manifest string) ([]map[string]interface{}, error) {
	ret := _m.Called(namespace, manifest)

	var r0 []map[string]interface{}
	if rf, ok := ret.Get(0).(func(string, string) []map[string]interface{}); ok {
		r0 = rf(namespace, manifest)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]map[string]interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, manifest)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ComputeReleaseResourcesByManifest provides a mock function with given fields: namespace, manifest
func (_m *Operator) ComputeReleaseResourcesByManifest(namespace string, manifest string) (*release.ReleaseResources, error) {
	ret := _m.Called(namespace, manifest)

	var r0 *release.ReleaseResources
	if rf, ok := ret.Get(0).(func(string, string) *release.ReleaseResources); ok {
		r0 = rf(namespace, manifest)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*release.ReleaseResources)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, manifest)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateLimitRange provides a mock function with given fields: limitRange
func (_m *Operator) CreateLimitRange(limitRange *modelsk8s.LimitRange) error {
	ret := _m.Called(limitRange)

	var r0 error
	if rf, ok := ret.Get(0).(func(*modelsk8s.LimitRange) error); ok {
		r0 = rf(limitRange)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateNamespace provides a mock function with given fields: namespace
func (_m *Operator) CreateNamespace(namespace *modelsk8s.Namespace) error {
	ret := _m.Called(namespace)

	var r0 error
	if rf, ok := ret.Get(0).(func(*modelsk8s.Namespace) error); ok {
		r0 = rf(namespace)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateOrUpdateResourceQuota provides a mock function with given fields: resourceQuota
func (_m *Operator) CreateOrUpdateResourceQuota(resourceQuota *modelsk8s.ResourceQuota) error {
	ret := _m.Called(resourceQuota)

	var r0 error
	if rf, ok := ret.Get(0).(func(*modelsk8s.ResourceQuota) error); ok {
		r0 = rf(resourceQuota)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateResourceQuota provides a mock function with given fields: resourceQuota
func (_m *Operator) CreateResourceQuota(resourceQuota *modelsk8s.ResourceQuota) error {
	ret := _m.Called(resourceQuota)

	var r0 error
	if rf, ok := ret.Get(0).(func(*modelsk8s.ResourceQuota) error); ok {
		r0 = rf(resourceQuota)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateSecret provides a mock function with given fields: namespace, secretRequestBody
func (_m *Operator) CreateSecret(namespace string, secretRequestBody *modelsk8s.CreateSecretRequestBody) error {
	ret := _m.Called(namespace, secretRequestBody)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *modelsk8s.CreateSecretRequestBody) error); ok {
		r0 = rf(namespace, secretRequestBody)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteIsomateSetPvcs provides a mock function with given fields: isomateSets
func (_m *Operator) DeleteIsomateSetPvcs(isomateSets []*modelsk8s.IsomateSet) error {
	ret := _m.Called(isomateSets)

	var r0 error
	if rf, ok := ret.Get(0).(func([]*modelsk8s.IsomateSet) error); ok {
		r0 = rf(isomateSets)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteNamespace provides a mock function with given fields: name
func (_m *Operator) DeleteNamespace(name string) error {
	ret := _m.Called(name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletePod provides a mock function with given fields: namespace, name
func (_m *Operator) DeletePod(namespace string, name string) error {
	ret := _m.Called(namespace, name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(namespace, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletePodMigration provides a mock function with given fields: namespace, name
func (_m *Operator) DeletePodMigration(namespace string, name string) error {
	ret := _m.Called(namespace, name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(namespace, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletePvc provides a mock function with given fields: namespace, name
func (_m *Operator) DeletePvc(namespace string, name string) error {
	ret := _m.Called(namespace, name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(namespace, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletePvcs provides a mock function with given fields: namespace, labelSeletorStr
func (_m *Operator) DeletePvcs(namespace string, labelSeletorStr string) error {
	ret := _m.Called(namespace, labelSeletorStr)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(namespace, labelSeletorStr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteSecret provides a mock function with given fields: namespace, name
func (_m *Operator) DeleteSecret(namespace string, name string) error {
	ret := _m.Called(namespace, name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(namespace, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteStatefulSetPvcs provides a mock function with given fields: statefulSets
func (_m *Operator) DeleteStatefulSetPvcs(statefulSets []*modelsk8s.StatefulSet) error {
	ret := _m.Called(statefulSets)

	var r0 error
	if rf, ok := ret.Get(0).(func([]*modelsk8s.StatefulSet) error); ok {
		r0 = rf(statefulSets)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LabelNode provides a mock function with given fields: name, labelsToAdd, labelsToRemove
func (_m *Operator) LabelNode(name string, labelsToAdd map[string]string, labelsToRemove []string) error {
	ret := _m.Called(name, labelsToAdd, labelsToRemove)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, map[string]string, []string) error); ok {
		r0 = rf(name, labelsToAdd, labelsToRemove)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MigrateNode provides a mock function with given fields: srcNode, destNode
func (_m *Operator) MigrateNode(srcNode string, destNode string) error {
	ret := _m.Called(srcNode, destNode)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(srcNode, destNode)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MigratePod provides a mock function with given fields: mig
func (_m *Operator) MigratePod(mig *modelsk8s.Mig) error {
	ret := _m.Called(mig)

	var r0 error
	if rf, ok := ret.Get(0).(func(*modelsk8s.Mig) error); ok {
		r0 = rf(mig)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RestartPod provides a mock function with given fields: namespace, name
func (_m *Operator) RestartPod(namespace string, name string) error {
	ret := _m.Called(namespace, name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(namespace, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TaintNoExecuteNode provides a mock function with given fields: name, taintsToAdd, taintsToRemove
func (_m *Operator) TaintNoExecuteNode(name string, taintsToAdd map[string]string, taintsToRemove []string) error {
	ret := _m.Called(name, taintsToAdd, taintsToRemove)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, map[string]string, []string) error); ok {
		r0 = rf(name, taintsToAdd, taintsToRemove)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateConfigMap provides a mock function with given fields: namespace, configMapName, requestBody
func (_m *Operator) UpdateConfigMap(namespace string, configMapName string, requestBody *modelsk8s.ConfigMapRequestBody) error {
	ret := _m.Called(namespace, configMapName, requestBody)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, *modelsk8s.ConfigMapRequestBody) error); ok {
		r0 = rf(namespace, configMapName, requestBody)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateIngress provides a mock function with given fields: namespace, ingressName, requestBody
func (_m *Operator) UpdateIngress(namespace string, ingressName string, requestBody *modelsk8s.IngressRequestBody) error {
	ret := _m.Called(namespace, ingressName, requestBody)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, *modelsk8s.IngressRequestBody) error); ok {
		r0 = rf(namespace, ingressName, requestBody)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateNamespace provides a mock function with given fields: namespace
func (_m *Operator) UpdateNamespace(namespace *modelsk8s.Namespace) error {
	ret := _m.Called(namespace)

	var r0 error
	if rf, ok := ret.Get(0).(func(*modelsk8s.Namespace) error); ok {
		r0 = rf(namespace)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateSecret provides a mock function with given fields: namespace, secretRequestBody
func (_m *Operator) UpdateSecret(namespace string, secretRequestBody *modelsk8s.CreateSecretRequestBody) error {
	ret := _m.Called(namespace, secretRequestBody)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *modelsk8s.CreateSecretRequestBody) error); ok {
		r0 = rf(namespace, secretRequestBody)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
