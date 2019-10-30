package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/migration/pkg/apis/tos/v1beta1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestConvertMigFromK8s(t *testing.T) {
	tests := []struct {
		oriMig *v1beta1.Mig
		mig    *k8s.Mig
		err    error
	}{
		{
			oriMig: &v1beta1.Mig{
				TypeMeta: metav1.TypeMeta{
					Kind: string(k8s.MigKind),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mig",
					Namespace: "test-namespace",
					Labels:    map[string]string{"test1": "test2"},
				},
				Spec: v1beta1.MigSpec{
					PodName:   "test-pod",
					Namespace: "test-namespace1",
				},
				Status: v1beta1.MigStatus{
					SrcHost:  "node01",
					DestHost: "node02",
					Phase: v1beta1.MIG_FINISH,
					ErrMsg: "",
				},
			},
			mig: &k8s.Mig{
				Meta: k8s.Meta{
					Name:      "test-mig",
					Namespace: "test-namespace",
					Kind:      k8s.MigKind,
					State:     k8s.State{
						Status:  "Finished",
						Message: "",
					},

				},
				Labels: map[string]string{"test1": "test2"},
				Spec: k8s.MigSpec{
					PodName:   "test-pod",
					Namespace: "test-namespace1",
				},
				SrcHost:  "node01",
				DestHost: "node02",
			},
			err: nil,
		},
		{
			oriMig: nil,
			mig:    nil,
			err:    errors.Errorf("oriMig is nil, invalid memory address or nil pointer dereference"),
		},
	}

	for _, test := range tests {
		mig, err := ConvertMigFromK8s(test.oriMig)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.mig, mig)
	}
}

func TestConvertMigToK8s(t *testing.T) {
	tests := []struct {
		oriMig *k8s.Mig
		k8sMig *v1beta1.Mig
		err    error
	}{
		{
			oriMig: &k8s.Mig{
				Meta: k8s.Meta{
					Name:      "test-mig",
					Namespace: "test-namespace",
					Kind:      k8s.MigKind,
				},
				Labels: map[string]string{"test1": "test2"},
				Spec: k8s.MigSpec{
					PodName:   "test-pod",
					Namespace: "test-namespace1",
				},
				SrcHost:  "node01",
				DestHost: "node02",
			},
			k8sMig: &v1beta1.Mig{
				TypeMeta: metav1.TypeMeta{
					Kind: string(k8s.MigKind),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mig",
					Namespace: "test-namespace",
					Labels:    map[string]string{"test1": "test2"},
				},
				Spec: v1beta1.MigSpec{
					PodName:   "test-pod",
					Namespace: "test-namespace1",
				},
				Status: v1beta1.MigStatus{
					SrcHost:  "node01",
					DestHost: "node02",
				},
			},
			err: nil,
		},
		{
			oriMig: &k8s.Mig{},
			k8sMig: &v1beta1.Mig{
				TypeMeta: metav1.TypeMeta{
					Kind: string(k8s.MigKind),
				},
			},
			err:    nil,
		},
	}

	for _, test := range tests {
		mig, err := ConvertMigToK8s(test.oriMig)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.k8sMig, mig)
	}
}
