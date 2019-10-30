package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func TestConvertReplicaSetFromK8s(t *testing.T) {
	var replicas int32 = 2
	var controller = true
	tests := []struct {
		oriReplicaSet *appsv1.ReplicaSet
		replicaSet    *k8s.ReplicaSet
		err           error
	}{
		{
			oriReplicaSet: &appsv1.ReplicaSet{
				TypeMeta: metav1.TypeMeta{
					Kind: string(k8s.ReplicaSetKind),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-replicaSet",
					Namespace: "test-namespace",
					UID:       types.UID("dfee574a-c7dc-11e9-91b6-d61387db2e94"),
					OwnerReferences: []metav1.OwnerReference{
						{
							Controller: &controller,
							Kind:       "Deployment",
							Name:       "minio",
							UID:        "dfecbb50-c7dc-11e9-83c3-ac1f6b83dd66",
						},
					},
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: &replicas,
				},
			},
			replicaSet: &k8s.ReplicaSet{
				Meta: k8s.Meta{
					Name:      "test-replicaSet",
					Namespace: "test-namespace",
					Kind:      "ReplicaSet",
				},
				UID:      "dfee574a-c7dc-11e9-91b6-d61387db2e94",
				Replicas: &replicas,
				OwnerReferences: []k8s.OwnerReference{
					{
						Controller: &controller,
						Kind:       "Deployment",
						Name:       "minio",
						UID:        "dfecbb50-c7dc-11e9-83c3-ac1f6b83dd66",
					},
				},
			},
		},
		{
			oriReplicaSet: nil,
			replicaSet:    nil,
			err:           errors.Errorf("oriReplicaSet is nil, invalid memory address or nil pointer dereference"),
		},
	}

	for _, test := range tests {
		replicaSet, err := ConvertReplicaSetFromK8s(test.oriReplicaSet)
		assert.IsType(t, test.err, err)
		assert.IsType(t, test.replicaSet, replicaSet)
	}
}
