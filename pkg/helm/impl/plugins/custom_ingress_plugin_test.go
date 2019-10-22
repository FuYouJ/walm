package plugins

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"k8s.io/apimachinery/pkg/runtime"
	"encoding/json"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"helm.sh/helm/pkg/release"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Test_CustomIngressTransform(t *testing.T) {
	tests := []struct {
		context *PluginContext
		args    *CustomIngressArgs
		err     error
		result  *PluginContext
	}{
		{
			context: &PluginContext{
				Resources: []runtime.Object{
					convertObjToUnstructured(&v1beta1.Ingress{
						TypeMeta: v1.TypeMeta{
							Kind: "Ingress",
						},
					}),
				},
				R: &release.Release{
					Namespace: "testns",
					Name: "testnm",
				},
			},
			args: &CustomIngressArgs{
				IngressSkipAll: true,
				IngressToAdd: map[string]*AddIngressObject{
					"adding": {
						Path: "/test-path",
						Host: "test-host",
						ServiceName: "test-svc",
						ServicePort: "test-port",
					},
				},
			},
			result: &PluginContext{
				Resources: []runtime.Object{
					convertObjToUnstructured(&v1beta1.Ingress{
						TypeMeta: v1.TypeMeta{
							Kind: "Ingress",
						},
						ObjectMeta: v1.ObjectMeta{
							Annotations: map[string]string{
								ResourceUpgradePolicyAnno: UpgradePolicy,
							},
						},
					}),
					convertObjToUnstructured(&v1beta1.Ingress{
						TypeMeta: v1.TypeMeta{
							Kind: "Ingress",
						},
						ObjectMeta: v1.ObjectMeta{
							Name:      fmt.Sprintf("walmplugin-%s-%s-ingress", "adding", "testnm"),
							Namespace: "testns",
							Annotations: map[string]string{
								"transwarp/walmplugin.custom.ingress": "true",
								"kubernetes.io/ingress.class":         "nginx",
							},
							Labels: map[string]string{
								"release":  "testnm",
								"heritage": "walmplugin",
							},
						},
						Spec: v1beta1.IngressSpec{
							Rules: []v1beta1.IngressRule{
								{
									Host: "test-host",
									IngressRuleValue: v1beta1.IngressRuleValue{
										HTTP: &v1beta1.HTTPIngressRuleValue{
											Paths: []v1beta1.HTTPIngressPath{
												{
													Path: "/test-path",
													Backend: v1beta1.IngressBackend{
														ServiceName: "test-svc",
														ServicePort: intstr.IntOrString{
															Type:   intstr.String,
															StrVal: "test-port",
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
				R: &release.Release{
					Namespace: "testns",
					Name: "testnm",
				},
			},
		},
	}

	for _, test := range tests {
		argsBytes, _ := json.Marshal(test.args)

		err := CustomIngressTransform(test.context, string(argsBytes))
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.result, test.context)
	}
}
