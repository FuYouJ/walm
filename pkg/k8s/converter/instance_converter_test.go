package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"transwarp/application-instance/pkg/apis/transwarp/v1beta1"
)

func TestConvertInstanceFromK8s(t *testing.T) {

	tests := []struct {
		oriInst        *v1beta1.ApplicationInstance
		instModules    *k8s.ResourceSet
		dependencyMeta *k8s.DependencyMeta
		inst           *k8s.ApplicationInstance
		err            error
	}{
		{
			oriInst:        nil,
			instModules:    nil,
			dependencyMeta: nil,
			inst:           nil,
			err:            nil,
		},
		{
			oriInst: &v1beta1.ApplicationInstance{
				TypeMeta: metav1.TypeMeta{
					Kind: "ApplicationInstance",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ka",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{},
				},
				Spec: v1beta1.ApplicationInstanceSpec{
					ApplicationRef: v1beta1.ApplicationReference{
						Name:    "zookeeper",
						Version: "5.2",
					},
					InstanceId: "f6b4t",
					Dependencies: []v1beta1.Dependency{
						{
							Name: "zookeeper",
							DependencyRef: v1.ObjectReference{
								Kind:      "ApplicationInstance",
								Namespace: "test-namespace",
								Name:      "test-zk",
							},
						},
						{
							Name: "txsql",
							DependencyRef: v1.ObjectReference{
								Kind:      "ApplicationInstance",
								Namespace: "test-namespace2",
								Name:      "test-txsql",
							},
						},
					},
				},
			},
			instModules:    nil,
			dependencyMeta: nil,
			inst: &k8s.ApplicationInstance{
				Meta: k8s.Meta{
					Name:      "test-ka",
					Namespace: "test-namespace",
					Kind:      k8s.InstanceKind,
					State: k8s.State{
						Status:  "Ready",
						Reason:  "",
						Message: "",
					},
				},
				CreationTimestamp: "0001-01-01 00:00:00 +0000 UTC",
				Dependencies:   map[string]string{"zookeeper": "test-zk", "txsql": "test-namespace2/test-txsql"},
				InstanceId:     "f6b4t",
				Modules:        nil,
				DependencyMeta: nil,
			},
		},
	}

	for _, test := range tests {
		inst, err := ConvertInstanceFromK8s(test.oriInst, test.instModules, test.dependencyMeta)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.inst, inst)
	}
}
