/*

Copyright 2019 Transwarp All rights reserved.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	ControllerRevisionHashLabelKey = "controller-revision-hash"
	IsomateSetRevisionLabel        = ControllerRevisionHashLabelKey
	IsomateSetPodNameLabel         = "isomateset.transwarp.io/pod-name"
	IsomateSetVersionNameLabel     = "isomateset.transwarp.io/pod-version"
	IsomateSetNameLabel            = "isomateset.transwarp.io/isomateset-name"
	IsomateSetOfflineAnnoKey       = "isomateset.transwarp.io/offline-state"

	IsomateSetVolumeStrategyAnnoKey = "isomateset.transwarp.io/volume-strategy"
)

const (
	// PreserveVolumeStrategy will preserve pvcs created by a pod when the pod was deleted
	PreserveVolumeStrategy = "Preserve"
	//OnDeleteVolumeStrategy wil delete pvcs created by a pod when the pod was deleted
	OnDeleteVolumeStrategy = "OnDelete"
)

// IsomateSetSpec defines the desired state of IsomateSet
type IsomateSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Selector for pods management
	Selector *metav1.LabelSelector `json:"selector"`
	// TotalReplicas is the total desired replicas of all the versioned Templates.
	// If unspecified, defaults to a value according to default policy.
	// +optional
	TotalReplicas *int32 `json:"totalReplicas,omitempty"`
	// VersionTemplates is a map recording the version and its corresponding pod template
	VersionTemplates map[string]*VersionTemplateSpec `json:"versionTemplates"`
	// Define a list of PVCs
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	// Define a list of indexs which Isomate will treat as offline pods index
	OfflineIndexes OfflineIndexes `json:"offlineIndexes,omitempty"`
	// AvailableIndexes is a map recording the version name and its corresponding available indexes
	// this field should not be modified by user.
	// AvailableIndexes AvailableIndexes `json:"availableIndexes,omitempty"`
	// Plugin handlers for customized operations
	Handlers []IsomateSetHandler `json:"handlers,omitempty"`
}

type IsomateSetHandler struct {
	// Handler name
	Name string `json:"name,omitempty"`
	// Handler function args
	Args string `json:"args,omitempty"`
}

type OfflineIndexes map[string]IndexList

// '[0,1,2]'
type IndexList []int
type VersionTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Replicas is the desired number of replicas of the given Template.
	// These are replicas in the sense that they are instantiations of the
	// same Template, but individual replicas also have a consistent identity.
	// If unspecified, defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Indicates the number of the pod to be created under this version. Replicas could also be
	// percentage like '10%', which means 10% of IsomateSet replicas of pods will be distributed
	// under this version. If nil, the number of replicas in this version is determined by controller.
	// Controller will try to keep all the versions with nil replicas have average pods.
	// +optional
	ExpectedReplicas *intstr.IntOrString `json:"expectedReplicas,omitempty"`

	// Template describes the pods that will be created
	Template v1.PodTemplateSpec `json:"template"`

	// serviceName is the name of the service that governs this isomateSet.
	// This service must exist before the isomateSet, and is responsible for
	// the network identity of the set. Pods get DNS/hostnames that follow the
	// pattern: pod-specific-string.serviceName.default.svc.cluster.local
	// where "pod-specific-string" is managed by the isomateSet controller.
	ServiceName string `json:"serviceName" protobuf:"bytes,5,opt,name=serviceName"`

	// podManagementPolicy controls how pods are created during initial scale up,
	// when replacing pods on nodes, or when scaling down. The default policy is
	// `OrderedReady`, where pods are created in increasing order (pod-0, then
	// pod-1, etc) and the controller will wait until each pod is ready before
	// continuing. When scaling down, the pods are removed in the opposite order.
	// The alternative policy is `Parallel` which will create pods in parallel
	// to match the desired scale without waiting, and on scale down will delete
	// all pods at once.
	// +optional
	PodManagementPolicy PodManagementPolicyType `json:"podManagementPolicy,omitempty" protobuf:"bytes,6,opt,name=podManagementPolicy,casttype=PodManagementPolicyType"`

	// VolumeStrategy that will be employed to create and delete PVCs.
	// +optional
	// TODO: valid the named pvc volume exist in VolumeClaimTemplates
	VolumeStrategy IsomateSetVolumeStrategy `json:"volumeStrategy,omitempty"`

	// UpdateStrategy that will be employed to update Pods
	UpdateStrategy IsomateSetUpdateStrategy `json:"updateStrategy,omitempty"`
	// revisionHistoryLimit is the maximum number of revisions that will
	// be maintained in the IsomateSet's revision history. The revision history
	// consists of all revisions not represented by a currently applied
	// IsomateSetSpec version. The default value is 10.
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty" protobuf:"varint,8,opt,name=revisionHistoryLimit"`
}

// PodManagementPolicyType defines the policy for creating pods under a stateful set.
type PodManagementPolicyType string

const (
	// OrderedReadyPodManagement will create pods in strictly increasing order on
	// scale up and strictly decreasing order on scale down, progressing only when
	// the previous pod is ready or terminated. At most one pod will be changed
	// at any time.
	OrderedReadyPodManagement PodManagementPolicyType = "OrderedReady"
	// ParallelPodManagement will create and delete pods as soon as the stateful set
	// replica count is changed, and will not wait for pods to be ready or complete
	// termination.
	ParallelPodManagement = "Parallel"
)

// IsomateSetVolumeStrategy defines the policy for deleting pvcs created by a pod
// when the pod was deleted.
type IsomateSetVolumeStrategy struct {
	// Type indicates the type of the IsomateSetVolumeStrategy, default is `Preserve`,
	// where we preserve pod PVCs when the pod was being deleted. The alternative
	// strategy is `OnDelete`, where we delete pod PVCs when the pod was being deleted.
	// +optional
	Type string `json:"type,omitempty"`
	// Define a list of PVC names that will be created when the corresponding pod is
	// being created. A vaild PVC name should be extracted from `VolumeClaimTemplates`
	// list, default is nil.
	// +optional
	Names []string `json:"names,omitempty"`
}

// IsomateSetUpdateStrategy indicates the strategy that the IsomateSet
// controller will use to perform updates. It includes any additional parameters
// necessary to perform the update for the indicated strategy.
type IsomateSetUpdateStrategy struct {
	// Type indicates the type of the IsomateSetUpdateStrategy.
	// Default is RollingUpdate.
	// +optional
	Type IsomateSetUpdateStrategyType `json:"type,omitempty"`
	// RollingUpdate is used to communicate parameters when Type is RollingUpdateIsomateSetStrategyType.
	// +optional
	RollingUpdate *RollingUpdateIsomateSetStrategy `json:"rollingUpdate,omitempty"`
}

// IsomateSetUpdateStrategyType is a string enumeration type that enumerates
// all possible update strategies for the IsomateSet controller.
type IsomateSetUpdateStrategyType string

const (
	// RollingUpdateIsomateSetStrategyType replaces the old pod by new one using rolling update
	// i.e gradually scale down the old pods and scale up the new one.
	RollingUpdateIsomateSetStrategyType IsomateSetUpdateStrategyType = "RollingUpdate"
	// OnDeleteIsomateSetStrategyType triggers the legacy behavior. Version
	// tracking and ordered rolling restarts are disabled. Pods are recreated
	// from the IsomateSetSpec when they are manually deleted. When a scale
	// operation is performed with this strategy,specification version indicated
	// by the IsomateSet's currentRevision.
	OnDeleteIsomateSetStrategyType = "OnDelete"
	// RecreateDeploymentStrategyType kills all existing pods before creating new ones.
	// RecreateDeploymentStrategyType IsomateSetUpdateStrategyType = "Recreate"
)

type RollingUpdateIsomateSetStrategy struct {
	// The maximum number of pods that can be unavailable during the update.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// Absolute number is calculated from percentage by rounding down.
	// This can not be 0 if MaxSurge is 0.
	// Defaults to 25%.
	// Example: when this is set to 30%, the old ReplicaSet can be scaled down to 70% of desired pods
	// immediately when the rolling update starts. Once new pods are ready, old ReplicaSet
	// can be scaled down further, followed by scaling up the new ReplicaSet, ensuring
	// that the total number of pods available at all times during the update is at
	// least 70% of desired pods.
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`

	// The maximum number of pods that can be scheduled above the desired number of
	// pods.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// This can not be 0 if MaxUnavailable is 0.
	// Absolute number is calculated from percentage by rounding up.
	// Defaults to 25%.
	// Example: when this is set to 30%, the new ReplicaSet can be scaled up immediately when
	// the rolling update starts, such that the total number of old and new pods do not exceed
	// 130% of desired pods. Once old pods have been killed,
	// new ReplicaSet can be scaled up further, ensuring that total number of pods running
	// at any time during the update is at most 130% of desired pods.
	// +optional
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty"`
	// Partition indicates the ordinal at which the IsomateSet should be
	// partitioned.
	Partition *int32 `json:"partition,omitempty" protobuf:"varint,1,opt,name=partition"`
}

// IsomateSetStatus defines the observed state of IsomateSet
type IsomateSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The most recent generation observed for this IsomateSet
	ObservedGeneration *int64 `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
	// Template status for each version
	VersionTemplateStatus map[string]*TemplateStatus `json:"versionTemplateStatus,omitempty"`
	// Represents the latest available observations of a isomateset's current state.
	Conditions []IsomateSetCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,10,rep,name=conditions"`
}

// TemplateStatus defines the observed state of each versioned template
type TemplateStatus struct {

	// replicas is the number of Pods created by the IsomateSet controller.
	Replicas int32 `json:"replicas" protobuf:"varint,2,opt,name=replicas"`

	// readyReplicas is the number of Pods created by the IsomateSet controller that have a Ready Condition.
	ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,3,opt,name=readyReplicas"`

	// currentReplicas is the number of Pods created by the IsomateSet controller from the IsomateSet version
	// indicated by currentRevision.
	CurrentReplicas int32 `json:"currentReplicas,omitempty" protobuf:"varint,4,opt,name=currentReplicas"`

	// updatedReplicas is the number of Pods created by the IsomateSet controller from the IsomateSet version
	// indicated by updateRevision.
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty" protobuf:"varint,5,opt,name=updatedReplicas"`

	// currentRevision, if not empty, indicates the version of the IsomateSet used to generate Pods in the
	// sequence [0,currentReplicas).
	CurrentRevision string `json:"currentRevision,omitempty" protobuf:"bytes,6,opt,name=currentRevision"`

	// updateRevision, if not empty, indicates the version of the IsomateSet used to generate Pods in the sequence
	// [replicas-updatedReplicas,replicas)
	UpdateRevision string `json:"updateRevision,omitempty" protobuf:"bytes,7,opt,name=updateRevision"`

	// collisionCount is the count of hash collisions for the IsomateSet. The IsomateSet controller
	// uses this field as a collision avoidance mechanism when it needs to create the name for the
	// newest ControllerRevision.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty" protobuf:"varint,9,opt,name=collisionCount"`
}

type IsomateSetConditionType string
type IsomateSetCondition struct {
	// Type of isomateSet condition.
	Type IsomateSetConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=IsomateSetConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
	// The current processing handler
	Handler string `json:"handler,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=isomatesets,scope=Namespaced,singular=isomateset,shortName=ims;isomate
// +kubebuilder:subresource:status
// IsomateSet is the Schema for the isomatesets API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type IsomateSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IsomateSetSpec   `json:"spec,omitempty"`
	Status IsomateSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IsomateSetList contains a list of IsomateSet
type IsomateSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IsomateSet `json:"items"`
}
