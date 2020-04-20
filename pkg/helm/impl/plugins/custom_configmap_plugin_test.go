package plugins

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

func Test_convertK8SConfigMap(t *testing.T) {
	releaseName := "test-release"
	releaseNamespace := "test"
	configMapName := "test-cm"
	addObj := &AddConfigmapObject{
		ApplyAllResources: true,
		Kind:              "",
		ResourceName:      "",
		ContainerName:     "",
		Items: []*AddConfigItem{
			{
				ConfigMapData:                  "test data\n",
				ConfigMapVolumeMountsMountPath: "/aa/bb/c",
				ConfigMapVolumeMountsSubPath:   "path-name",
			},
		},
	}

	configMap, _ := convertK8SConfigMap(releaseName, releaseNamespace, configMapName, addObj)
	assert.Equal(t, configMap.Kind, "ConfigMap")
	assert.Equal(t, configMap.Data, map[string]string{"/aa/bb/c/path-name": "test data\n"})
}

func Test_mountConfigMap(t *testing.T) {
	tests := []struct {
		unstructuredObj *unstructured.Unstructured
		releaseName     string
		configMapName   string
		addConfigMapObj *AddConfigmapObject
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
								},
							},
						},
					},
				},
			}),
			releaseName: "test-rel",
			configMapName: "test-cm",
			addConfigMapObj: &AddConfigmapObject{
				ApplyAllResources: true,
				Items: []*AddConfigItem{
					{
						ConfigMapVolumeMountsMountPath: "first",
						ConfigMapVolumeMountsSubPath: "second",
					},
				},
			},
			result: convertObjToUnstructured(&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container",
									VolumeMounts: []v1.VolumeMount{
										{
											Name:      "walmplugin-test-cm-test-rel-cm",
											MountPath: "first",
											SubPath:   "second",
										},
									},
								},
							},
							Volumes: []v1.Volume{
								{
									Name: "walmplugin-test-cm-test-rel-cm",
									VolumeSource: v1.VolumeSource{
										ConfigMap: &v1.ConfigMapVolumeSource{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "walmplugin-test-cm-test-rel-cm",
											},
											Items: []v1.KeyToPath{
												{
													Key:  "second",
													Path: "second",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}),
		},
	}

	for _, test := range tests {
		err := mountConfigMap(test.unstructuredObj, test.releaseName, test.configMapName, test.addConfigMapObj)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.result, test.unstructuredObj)
	}
}