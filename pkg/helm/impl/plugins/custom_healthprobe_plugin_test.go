package plugins

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

func Test_customHealthProbe(t *testing.T) {
	tests := []struct {
		unstructuredObj *unstructured.Unstructured
		customHealthProbeArgs *CustomHealthProbeArgs
		err             error
		result          *unstructured.Unstructured
	}{
		{
			unstructuredObj: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
									LivenessProbe: &v1.Probe{
									},
									ReadinessProbe: &v1.Probe{},
								},
							},
						},
					},
				},
			}),
			customHealthProbeArgs: &CustomHealthProbeArgs{
				DisableAllReadinessProbe: true,
				DisableAllLivenessProbe: true,
			},
			result: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
								},
							},
						},
					},
				},
			}),
		},
	}

	for _, test := range tests {
		err := customHealthProbe(test.unstructuredObj, test.customHealthProbeArgs)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.result, test.unstructuredObj)
	}
}
