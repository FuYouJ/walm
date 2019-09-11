package elect

import (
	"context"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"os"
	"time"
)

type Elector struct {
	elector *leaderelection.LeaderElector
}

type ElectorConfig struct {
	Client               *kubernetes.Clientset
	ElectionId           string
	LockNamespace        string
	LockIdentity         string
	OnStartedLeadingFunc func(context context.Context)
	OnStoppedLeadingFunc func()
	OnNewLeaderFunc      func(identity string)
}

func (elector *Elector) Run(context context.Context) {
	elector.elector.Run(context)
}

func (elector *Elector) IsLeader() bool {
	return elector.elector.IsLeader()
}

func (elector *Elector) GetLeader() string {
	return elector.elector.GetLeader()
}

func NewElector(config *ElectorConfig) (*Elector, error) {
	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: config.OnStartedLeadingFunc,
		OnStoppedLeading: config.OnStoppedLeadingFunc,
		OnNewLeader:      config.OnNewLeaderFunc,
	}

	broadcaster := record.NewBroadcaster()
	hostname, _ := os.Hostname()

	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{
		Component: "walm-leader-elector",
		Host:      hostname,
	})

	lock := resourcelock.ConfigMapLock{
		ConfigMapMeta: metav1.ObjectMeta{Namespace: config.LockNamespace, Name: config.ElectionId},
		Client:        config.Client.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      config.LockIdentity,
			EventRecorder: recorder,
		},
	}

	ttl := 30 * time.Second
	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          &lock,
		LeaseDuration: ttl,
		RenewDeadline: ttl / 2,
		RetryPeriod:   ttl / 4,
		Callbacks:     callbacks,
	})
	if err != nil {
		klog.Error("failed to new leader elector")
		return nil, err
	}
	return &Elector{le}, nil
}
