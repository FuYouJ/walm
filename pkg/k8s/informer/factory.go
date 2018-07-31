package informer

import (
	"k8s.io/client-go/informers"
	"transwarp/application-instance/pkg/client/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	clientsetex "transwarp/application-instance/pkg/client/clientset/versioned"
	"time"
	listv1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	tranv1beta1 "transwarp/application-instance/pkg/client/listers/transwarp/v1beta1"
	"k8s.io/client-go/listers/core/v1"
	batchv1 "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/listers/apps/v1beta1"
	"walm/pkg/k8s/client"
	"k8s.io/apimachinery/pkg/util/wait"
)

var Factory InformerFactory

func init() {
	Factory = NewInformerFactory(client.GetDefaultClient(), client.GetDefaultClientEx(), 0)
	Factory.Start(wait.NeverStop)
	Factory.WaitForCacheSync(wait.NeverStop)
}

type InformerFactory struct {
	Factory informers.SharedInformerFactory
	DeploymentLister listv1beta1.DeploymentLister
	ConfigMapLister v1.ConfigMapLister
	DaemonSetLister listv1beta1.DaemonSetLister
	IngressLister listv1beta1.IngressLister
	JobLister batchv1.JobLister
	PodLister v1.PodLister
	SecretLister v1.SecretLister
	ServiceLister v1.ServiceLister
	StatefulSetLister v1beta1.StatefulSetLister
	NodeLister v1.NodeLister
	NamespaceLister v1.NamespaceLister

	FactoryEx externalversions.SharedInformerFactory
	InstanceLister tranv1beta1.ApplicationInstanceLister
}

func (factory InformerFactory) Start(stopCh <-chan struct{}) {
	factory.Factory.Start(stopCh)
	factory.FactoryEx.Start(stopCh)
}

func (factory InformerFactory) WaitForCacheSync(stopCh <-chan struct{}) {
	factory.Factory.WaitForCacheSync(stopCh)
	factory.FactoryEx.WaitForCacheSync(stopCh)
}

func NewInformerFactory(client *kubernetes.Clientset, clientEx *clientsetex.Clientset, resyncPeriod time.Duration) (InformerFactory) {
	factory := InformerFactory{}
	factory.Factory =  informers.NewSharedInformerFactory(client, resyncPeriod)
	factory.DeploymentLister = factory.Factory.Extensions().V1beta1().Deployments().Lister()
	factory.ConfigMapLister = factory.Factory.Core().V1().ConfigMaps().Lister()
	factory.DaemonSetLister = factory.Factory.Extensions().V1beta1().DaemonSets().Lister()
	factory.IngressLister = factory.Factory.Extensions().V1beta1().Ingresses().Lister()
	factory.JobLister = factory.Factory.Batch().V1().Jobs().Lister()
	factory.PodLister = factory.Factory.Core().V1().Pods().Lister()
	factory.SecretLister = factory.Factory.Core().V1().Secrets().Lister()
	factory.ServiceLister = factory.Factory.Core().V1().Services().Lister()
	factory.StatefulSetLister = factory.Factory.Apps().V1beta1().StatefulSets().Lister()
	factory.NodeLister = factory.Factory.Core().V1().Nodes().Lister()
	factory.NamespaceLister = factory.Factory.Core().V1().Namespaces().Lister()

	factory.FactoryEx = externalversions.NewSharedInformerFactory(clientEx, resyncPeriod)
	factory.InstanceLister = factory.FactoryEx.Transwarp().V1beta1().ApplicationInstances().Lister()
	return factory
}