package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConvertReplicaSetFromK8s(oriReplicaSet *appsv1.ReplicaSet) (walmReplicaSet *k8s.ReplicaSet, err error) {

	if oriReplicaSet == nil {
		return nil, nil
	}
	replicaSet := oriReplicaSet.DeepCopy()

	walmReplicaSet = &k8s.ReplicaSet{
		Meta:     k8s.NewEmptyStateMeta(k8s.ReplicaSetKind, replicaSet.Namespace, replicaSet.Name),
		UID:      string(replicaSet.UID),
		Replicas: replicaSet.Spec.Replicas,
		Labels:   replicaSet.Labels,
		OwnerReferences: []k8s.OwnerReference{
			{

			},
		},
		Status: k8s.ReplicaSetStatus{
			Replicas:             replicaSet.Status.Replicas,
			FullyLabeledReplicas: replicaSet.Status.FullyLabeledReplicas,
			ReadyReplicas:        replicaSet.Status.ReadyReplicas,
			AvailableReplicas:    replicaSet.Status.AvailableReplicas,
			ObservedGeneration:   replicaSet.Status.ObservedGeneration,
		},
	}

	walmReplicaSet.OwnerReferences = buildWalmOwnerRef(replicaSet.OwnerReferences)
	walmReplicaSet.Status.Conditions = buildWalmReplicaSetConditons(replicaSet.Status.Conditions)
	return
}

func buildWalmReplicaSetConditons(conditions []appsv1.ReplicaSetCondition) []k8s.ReplicaSetCondition {
	var walmRsConditions []k8s.ReplicaSetCondition
	for _, condition := range conditions {
		walmRsConditions = append(walmRsConditions, k8s.ReplicaSetCondition{
			Type:    string(condition.Type),
			Status:  string(condition.Status),
			Reason:  condition.Reason,
			Message: condition.Message,
		})
	}
	return walmRsConditions
}

func buildWalmOwnerRef(ownerRefs []v1.OwnerReference) []k8s.OwnerReference {
	var walmOwnerRefs []k8s.OwnerReference
	for _, ownerRef := range ownerRefs {
		walmOwnerRefs = append(walmOwnerRefs, k8s.OwnerReference{
			Kind:       ownerRef.Kind,
			Name:       ownerRef.Name,
			UID:        string(ownerRef.UID),
			Controller: ownerRef.Controller,
		})
	}
	return walmOwnerRefs
}
