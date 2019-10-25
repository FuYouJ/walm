package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/migration/pkg/apis/tos/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
func ConvertMigToK8s(mig *k8s.Mig) (*v1beta1.Mig, error) {

	return &v1beta1.Mig{
		TypeMeta: metav1.TypeMeta{
			Kind: string(k8s.MigKind),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       mig.Name,
			Namespace:                  mig.Namespace,
			Labels:                     mig.Labels,
		},
		Spec: v1beta1.MigSpec{
			PodName:    mig.Spec.PodName,
			Namespace:  mig.Spec.Namespace,
		},
		Status: v1beta1.MigStatus{
			SrcHost:              mig.SrcHost,
			DestHost:             mig.DestHost,
		},
	}, nil
}

func ConvertMigFromK8s(oriMig *v1beta1.Mig) (*k8s.Mig, error) {
	if oriMig == nil {
		return nil, errors.Errorf("oriMig is nil, invalid memory address or nil pointer dereference")
	}
	mig := oriMig.DeepCopy()

	return &k8s.Mig{
		Meta:     k8s.Meta{
			Namespace: mig.Namespace,
			Name: mig.Name,
			Kind: k8s.MigKind,
			State: k8s.State{
				Status:  string(mig.Status.Phase),
				Message: mig.Status.ErrMsg,
			},
		},
		Labels:   mig.Labels,
		Spec:     k8s.MigSpec{
			PodName: mig.Spec.PodName,
			Namespace: mig.Spec.Namespace,
		},
		SrcHost:  mig.Status.SrcHost,
		DestHost: mig.Status.DestHost,
	}, nil
}
