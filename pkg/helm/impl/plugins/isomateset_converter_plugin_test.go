package plugins

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/core/v1"
	"transwarp/isomateset-client/pkg/apis/apiextensions.transwarp.io/v1alpha1"
)

func Test_ConvertStsToIsomateSet(t *testing.T) {
	tests := []struct {
		context *PluginContext
		result  *PluginContext
		err     error
	}{
		{
			context: &PluginContext{
				Resources: []runtime.Object{
					convertObjToUnstructured(&appsv1.StatefulSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-sts1",
							Annotations: map[string]string{
								ConvertToIsoamteSetLabelKey: ConvertToIsoamteSetLabelValue,
							},
						},
						Spec: appsv1.StatefulSetSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"test": "test"},
							},
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name: "test-con",
								},
							},
						},
					}),
				},
			},
			result: &PluginContext{
				Resources: []runtime.Object{
					convertObjToUnstructured(&v1alpha1.IsomateSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-sts1",
						},
						TypeMeta: metav1.TypeMeta{
							Kind: "IsomateSet",
							APIVersion: "apiextensions.transwarp.io/v1alpha1",
						},
						Spec: v1alpha1.IsomateSetSpec{
							VersionTemplates: map[string]*v1alpha1.VersionTemplateSpec{
								"test-sts1": {
									ObjectMeta: metav1.ObjectMeta{
										Annotations: map[string]string{
											ConvertToIsoamteSetLabelKey: ConvertToIsoamteSetLabelValue,
										},
										Labels: map[string]string{
											"isomateset.transwarp.io/pod-version": "test-sts1",
										},
									},
									Template: v1.PodTemplateSpec{
										ObjectMeta: metav1.ObjectMeta{
											Name: "test-con",
										},
									},
								},
							},
							Selector:  &metav1.LabelSelector{
								MatchLabels: map[string]string{"isomateset.transwarp.io/isomateset-name": "test-sts1"},
							},
						},
					}),
				},
			},
		},
	}

	for _, test := range tests {
		err := ConvertStsToIsomateSet(test.context, "")
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.result, test.context)
	}

}
