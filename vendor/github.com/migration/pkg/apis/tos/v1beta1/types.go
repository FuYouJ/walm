package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Mig is a specification for a Mig resource
type Mig struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec MigSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// +optional
	Status MigStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

const (
	// Mig status
	MIG_CREATED     = "Created"
	MIG_IN_PROGRESS = "InProgress"

	MIG_FINISH = "Finished"
	MIG_FAILED = "Failed"

	// exponential backoff
	InitialDurationBeforeRetry = 500 * time.Millisecond
	MaxDurationBeforeRetry     = 2*time.Minute + 2*time.Second
)

type MigrationStatus string

// MigSpec defines the desired state of Mig
type MigSpec struct {
	PodName    string `json:"podname,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	OfflinePod bool   `json:"offlinepod,omitempty"`
}

// MigStatus defines the observed state of Mig
type MigStatus struct {
	Phase MigrationStatus `json:"phase,omitempty"`
	// for showing information of Mig
	SrcHost              string        `json:"srcHost,omitempty"`
	DestHost             string        `json:"destHost,omitempty"`
	LastErrorTime        time.Time     `json:"lastErrorTime,omitempty"`
	DurationBeforeRetry  time.Duration `json:"durationBeforeRetry,omitempty"`
	LastBackOffCheckTime time.Time     `json:"lastBackOffCheckTime,omitempty"`
	ErrMsg               string        `json:"errMsg,omitempty"`
}

type BackOffError struct {
	Msg string
}

func (b BackOffError) Error() string {
	return b.Msg
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MigList is a list of Mig resources
type MigList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Mig `json:"items" protobuf:"bytes,2,rep,name=items"`
}
