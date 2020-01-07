package informer

import (
	"WarpCloud/walm/pkg/k8s/converter"
	"WarpCloud/walm/pkg/k8s/utils"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"errors"
	tosv1beta1 "github.com/migration/pkg/apis/tos/v1beta1"
	migrationclientset "github.com/migration/pkg/client/clientset/versioned"
	migrationexternalversions "github.com/migration/pkg/client/informers/externalversions"
	migrationv1beta1 "github.com/migration/pkg/client/listers/tos/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/listers/apps/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/apps/v1beta1"
	batchv1 "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/listers/core/v1"
	listv1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	storagev1 "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sort"
	"sync"
	"time"
	instanceclientset "transwarp/application-instance/pkg/client/clientset/versioned"
	instanceexternalversions "transwarp/application-instance/pkg/client/informers/externalversions"
	instancev1beta1 "transwarp/application-instance/pkg/client/listers/transwarp/v1beta1"
	releaseconfigclientset "transwarp/release-config/pkg/client/clientset/versioned"
	releaseconfigexternalversions "transwarp/release-config/pkg/client/informers/externalversions"
	releaseconfigv1beta1 "transwarp/release-config/pkg/client/listers/transwarp/v1beta1"

	k8sutils "WarpCloud/walm/pkg/k8s/utils"
	"fmt"
	beta1 "transwarp/application-instance/pkg/apis/transwarp/v1beta1"
)

type Informer struct {
	client                      *kubernetes.Clientset
	factory                     informers.SharedInformerFactory
	deploymentLister            listv1beta1.DeploymentLister
	configMapLister             v1.ConfigMapLister
	daemonSetLister             listv1beta1.DaemonSetLister
	ingressLister               listv1beta1.IngressLister
	jobLister                   batchv1.JobLister
	podLister                   v1.PodLister
	secretLister                v1.SecretLister
	serviceLister               v1.ServiceLister
	statefulSetLister           v1beta1.StatefulSetLister
	nodeLister                  v1.NodeLister
	namespaceLister             v1.NamespaceLister
	resourceQuotaLister         v1.ResourceQuotaLister
	persistentVolumeClaimLister v1.PersistentVolumeClaimLister
	storageClassLister          storagev1.StorageClassLister
	endpointsLister             v1.EndpointsLister
	limitRangeLister            v1.LimitRangeLister
	replicaSetLister            appsv1.ReplicaSetLister
	releaseConifgFactory releaseconfigexternalversions.SharedInformerFactory
	releaseConfigLister  releaseconfigv1beta1.ReleaseConfigLister

	instanceFactory instanceexternalversions.SharedInformerFactory
	instanceLister  instancev1beta1.ApplicationInstanceLister

	migrationFactory migrationexternalversions.SharedInformerFactory
	migrationLister  migrationv1beta1.MigLister
}

func (informer *Informer) ListServices(namespace string, labelSelectorStr string) ([]*k8s.Service, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}

	resources, err := informer.serviceLister.List(selector)
	if err != nil {
		klog.Errorf("failed to list services in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	services := []*k8s.Service{}
	for _, resource := range resources {
		endpoints, err := informer.getEndpoints(namespace, resource.Name)
		if err != nil && !errorModel.IsNotFoundError(err) {
			return nil, err
		}
		service, err := converter.ConvertServiceFromK8s(resource, endpoints)
		if err != nil {
			klog.Errorf("failed to convert service %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		services = append(services, service)
	}
	return services, nil
}

func (informer *Informer) ListStorageClasses(namespace string, labelSelectorStr string) ([]*k8s.StorageClass, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}

	resources, err := informer.storageClassLister.List(selector)
	if err != nil {
		klog.Errorf("failed to list storage classes in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	storageClasses := []*k8s.StorageClass{}
	for _, resource := range resources {
		storageClass, err := converter.ConvertStorageClassFromK8s(resource)
		if err != nil {
			klog.Errorf("failed to convert storageClass %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		storageClasses = append(storageClasses, storageClass)
	}
	return storageClasses, nil
}

func (informer *Informer) GetPodLogs(namespace string, podName string, containerName string, tailLines int64) (string, error) {
	podLogOptions := &corev1.PodLogOptions{}
	if containerName != "" {
		podLogOptions.Container = containerName
	}
	if tailLines != 0 {
		podLogOptions.TailLines = &tailLines
	}
	logs, err := informer.client.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions).Do().Raw()
	if err != nil {
		klog.Errorf("failed to get pod logs : %s", err.Error())
		return "", err
	}
	return string(logs), nil
}

func (informer *Informer) GetPodEventList(namespace string, name string) (*k8s.EventList, error) {
	pod, err := informer.podLister.Pods(namespace).Get(name)
	if err != nil {
		klog.Errorf("failed to get pod : %s", err.Error())
		return nil, err
	}

	ref := &corev1.ObjectReference{
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Kind:            pod.Kind,
		ResourceVersion: pod.ResourceVersion,
		UID:             pod.UID,
		APIVersion:      pod.APIVersion,
	}

	podEvents, err := informer.searchEvents(pod.Namespace, ref)
	if err != nil {
		klog.Errorf("failed to get Events : %s", err.Error())
		return nil, err
	}
	sort.Sort(utils.SortableEvents(podEvents.Items))

	walmEvents := []k8s.Event{}
	for _, event := range podEvents.Items {
		walmEvent := k8s.Event{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Count:          event.Count,
			FirstTimestamp: event.FirstTimestamp.String(),
			LastTimestamp:  event.LastTimestamp.String(),
			From:           utils.FormatEventSource(event.Source),
		}
		walmEvents = append(walmEvents, walmEvent)
	}
	return &k8s.EventList{Events: walmEvents}, nil
}

func (informer *Informer) GetDeploymentEventList(namespace string, name string) (*k8s.EventList, error) {
	deployment, err := informer.deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		klog.Errorf("failed to get deployment : %s", err.Error())
		return nil, err
	}

	ref := &corev1.ObjectReference{
		Kind:            deployment.Kind,
		Namespace:       deployment.Namespace,
		Name:            deployment.Name,
		UID:             deployment.UID,
		APIVersion:      deployment.APIVersion,
		ResourceVersion: deployment.ResourceVersion,
	}
	deploymentEvents, err := informer.searchEvents(deployment.Namespace, ref)
	if err != nil {
		klog.Errorf("failed to get Events : %s", err.Error())
		return nil, err
	}
	sort.Sort(utils.SortableEvents(deploymentEvents.Items))

	walmEvents := []k8s.Event{}
	for _, event := range deploymentEvents.Items {
		walmEvent := k8s.Event{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Count:          event.Count,
			FirstTimestamp: event.FirstTimestamp.String(),
			LastTimestamp:  event.LastTimestamp.String(),
			From:           utils.FormatEventSource(event.Source),
		}
		walmEvents = append(walmEvents, walmEvent)
	}
	return &k8s.EventList{Events: walmEvents}, nil
}

func (informer *Informer) GetStatefulSetEventList(namespace string, name string) (*k8s.EventList, error) {
	statefulSet, err := informer.statefulSetLister.StatefulSets(namespace).Get(name)
	if err != nil {
		klog.Errorf("failed to get statefulSet : %s", err.Error())
		return nil, err
	}

	ref := &corev1.ObjectReference{
		Kind:            statefulSet.Kind,
		Namespace:       statefulSet.Namespace,
		Name:            statefulSet.Name,
		UID:             statefulSet.UID,
		APIVersion:      statefulSet.APIVersion,
		ResourceVersion: statefulSet.ResourceVersion,
	}
	statefulSetEvents, err := informer.searchEvents(statefulSet.Namespace, ref)
	if err != nil {
		klog.Errorf("failed to get Events : %s", err.Error())
		return nil, err
	}
	sort.Sort(utils.SortableEvents(statefulSetEvents.Items))

	walmEvents := []k8s.Event{}
	for _, event := range statefulSetEvents.Items {
		walmEvent := k8s.Event{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Count:          event.Count,
			FirstTimestamp: event.FirstTimestamp.String(),
			LastTimestamp:  event.LastTimestamp.String(),
			From:           utils.FormatEventSource(event.Source),
		}
		walmEvents = append(walmEvents, walmEvent)
	}
	return &k8s.EventList{Events: walmEvents}, nil
}

func (informer *Informer) ListSecrets(namespace string, labelSelectorStr string) (*k8s.SecretList, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}

	resources, err := informer.secretLister.Secrets(namespace).List(selector)
	if err != nil {
		klog.Errorf("failed to list secrets in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	secrets := []*k8s.Secret{}
	for _, resource := range resources {
		secret, err := converter.ConvertSecretFromK8s(resource)
		if err != nil {
			klog.Errorf("failed to convert secret %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		secrets = append(secrets, secret)
	}
	return &k8s.SecretList{
		Num:   len(secrets),
		Items: secrets,
	}, nil
}

func (informer *Informer) ListStatefulSets(namespace string, labelSelectorStr string) ([]*k8s.StatefulSet, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}
	resources, err := informer.statefulSetLister.StatefulSets(namespace).List(selector)
	if err != nil {
		klog.Errorf("failed to list stateful sets in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	statefulSets := []*k8s.StatefulSet{}
	for _, resource := range resources {
		pods, err := informer.listPods(namespace, resource.Spec.Selector, false)
		if err != nil {
			return nil, err
		}
		statefulSet, err := converter.ConvertStatefulSetFromK8s(resource, pods)
		if err != nil {
			klog.Errorf("failed to convert stateful set %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		statefulSets = append(statefulSets, statefulSet)
	}
	return statefulSets, nil
}

func (informer *Informer) GetNodes(labelSelectorStr string) ([]*k8s.Node, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}
	nodeList, err := informer.nodeLister.List(selector)
	if err != nil {
		return nil, err
	}

	walmNodes := []*k8s.Node{}
	if nodeList != nil {
		mux := &sync.Mutex{}
		var wg sync.WaitGroup
		for _, node := range nodeList {
			wg.Add(1)
			go func(node *corev1.Node) {
				defer wg.Done()
				podsOnNode, err1 := informer.getNonTermiatedPodsOnNode(node.Name, nil)
				if err1 != nil {
					klog.Errorf("failed to get pods on node: %s", err1.Error())
					err = errors.New(err1.Error())
					return
				}
				walmNode, err1 := converter.ConvertNodeFromK8s(node, podsOnNode)
				if err1 != nil {
					klog.Errorf("failed to build walm node : %s", err1.Error())
					err = errors.New(err1.Error())
					return
				}

				mux.Lock()
				walmNodes = append(walmNodes, walmNode)
				mux.Unlock()
			}(node)
		}
		wg.Wait()
		if err != nil {
			klog.Errorf("failed to build nodes : %s", err.Error())
			return nil, err
		}
	}

	return walmNodes, nil
}

func (informer *Informer) AddReleaseConfigHandler(OnAdd func(obj interface{}), OnUpdate func(oldObj, newObj interface{}), OnDelete func(obj interface{})) {
	handlerFuncs := &cache.ResourceEventHandlerFuncs{
		AddFunc:    OnAdd,
		UpdateFunc: OnUpdate,
		DeleteFunc: OnDelete,
	}
	informer.releaseConifgFactory.Transwarp().V1beta1().ReleaseConfigs().Informer().AddEventHandler(handlerFuncs)
}

func (informer *Informer) AddServiceHandler(OnAdd func(obj interface{}), OnUpdate func(oldObj, newObj interface{}), OnDelete func(obj interface{})) {
	handlerFuncs := &cache.ResourceEventHandlerFuncs{
		AddFunc:    OnAdd,
		UpdateFunc: OnUpdate,
		DeleteFunc: OnDelete,
	}
	informer.factory.Core().V1().Services().Informer().AddEventHandler(handlerFuncs)
}

func (informer *Informer) AddMigrationHandler(OnAdd func(obj interface{}), OnUpdate func(oldObj, newObj interface{}), OnDelete func(obj interface{})) {

}
func (informer *Informer) ListPersistentVolumeClaims(namespace string, labelSelectorStr string) ([]*k8s.PersistentVolumeClaim, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}
	resources, err := informer.persistentVolumeClaimLister.PersistentVolumeClaims(namespace).List(selector)
	if err != nil {
		klog.Errorf("failed to list pvcs in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	pvcs := []*k8s.PersistentVolumeClaim{}
	for _, resource := range resources {
		pvc, err := converter.ConvertPvcFromK8s(resource)
		if err != nil {
			klog.Errorf("failed to convert release config %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		pvcs = append(pvcs, pvc)
	}
	return pvcs, nil
}

func (informer *Informer) ListReleaseConfigs(namespace, labelSelectorStr string) ([]*k8s.ReleaseConfig, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}
	resources, err := informer.releaseConfigLister.ReleaseConfigs(namespace).List(selector)
	if err != nil {
		klog.Errorf("failed to list release configs in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	releaseConfigs := []*k8s.ReleaseConfig{}
	for _, resource := range resources {
		releaseConfig, err := converter.ConvertReleaseConfigFromK8s(resource)
		if err != nil {
			klog.Errorf("failed to convert release config %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		releaseConfigs = append(releaseConfigs, releaseConfig)
	}
	return releaseConfigs, nil
}

func (informer *Informer) GetNodeMigration(namespace, node string) (*k8s.MigStatus, error) {
	var migs []*k8s.Mig
	selector, err := utils.ConvertLabelSelectorToSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{"migType": "node", "srcNode": node},
	})
	if err != nil {
		klog.Errorf("failed to convert label selector to selector: %s", err.Error())
		return nil, err
	}
	k8sMigs, err := informer.migrationLister.Migs(namespace).List(selector)
	if err != nil {
		klog.Errorf("failed to list pod migs of node: %s", err.Error())
		return nil, err
	}

	count := 0
	for _, k8sMig := range k8sMigs {
		if k8sMig.Status.Phase == tosv1beta1.MIG_FINISH {
			count++
		}
		mig, err := converter.ConvertMigFromK8s(k8sMig)
		if err != nil {
			klog.Errorf("failed to convert mig from k8s mig: %s", err.Error())
			return nil, err
		}

		migs = append(migs, mig)
	}
	return &k8s.MigStatus{
		Succeed: count,
		Total: len(k8sMigs),
		Items: migs,
	}, nil
}


func (informer *Informer) ListMigrations(namespace, labelSelectorStr string) ([]*k8s.Mig, error) {
	var k8sMigs []*tosv1beta1.Mig
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		klog.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}
	k8sMigs, err = informer.migrationLister.Migs(namespace).List(selector)

	var migs []*k8s.Mig
	for _, k8sMig := range k8sMigs {
		mig, err := converter.ConvertMigFromK8s(k8sMig)
		if err != nil {
			klog.Errorf("failed to convert mig from k8s: %s", err.Error())
			return nil, err
		}
		migs = append(migs, mig)
	}

	return migs, nil
}

func (informer *Informer) GetResourceSet(releaseResourceMetas []release.ReleaseResourceMeta) (resourceSet *k8s.ResourceSet, err error) {
	resourceSet = k8s.NewResourceSet()
	for _, resourceMeta := range releaseResourceMetas {
		resource, err := informer.GetResource(resourceMeta.Kind, resourceMeta.Namespace, resourceMeta.Name)
		// if resource is not found , do not return error, add it into resource set, so resource should not be nil
		if err != nil && !errorModel.IsNotFoundError(err) {
			return nil, err
		}
		resource.AddToResourceSet(resourceSet)
	}
	return
}

func (informer *Informer) GetResource(kind k8s.ResourceKind, namespace, name string) (k8s.Resource, error) {
	switch kind {
	case k8s.ReleaseConfigKind:
		return informer.getReleaseConfig(namespace, name)
	case k8s.ConfigMapKind:
		return informer.getConfigMap(namespace, name)
	case k8s.PersistentVolumeClaimKind:
		return informer.getPvc(namespace, name)
	case k8s.DaemonSetKind:
		return informer.getDaemonSet(namespace, name)
	case k8s.DeploymentKind:
		return informer.getDeployment(namespace, name)
	case k8s.ServiceKind:
		return informer.getService(namespace, name)
	case k8s.StatefulSetKind:
		return informer.getStatefulSet(namespace, name)
	case k8s.JobKind:
		return informer.getJob(namespace, name)
	case k8s.IngressKind:
		return informer.getIngress(namespace, name)
	case k8s.SecretKind:
		return informer.getSecret(namespace, name)
	case k8s.NodeKind:
		return informer.getNode(namespace, name)
	case k8s.StorageClassKind:
		return informer.getStorageClass(namespace, name)
	case k8s.InstanceKind:
		return informer.getInstance(namespace, name)
	case k8s.ReplicaSetKind:
		return informer.getReplicaSet(namespace, name)
	case k8s.MigKind:
		return informer.getMigration(namespace, name)
	default:
		return &k8s.DefaultResource{Meta: k8s.NewMeta(kind, namespace, name, k8s.NewState("Unknown", "NotSupportedKind", "Can not get this resource"))}, nil
	}
}

func (informer *Informer) start(stopCh <-chan struct{}) {
	informer.factory.Start(stopCh)
	informer.releaseConifgFactory.Start(stopCh)
	if informer.instanceFactory != nil {
		informer.instanceFactory.Start(stopCh)
	}
	if informer.migrationFactory != nil {
		informer.migrationFactory.Start(stopCh)
	}
}

func (informer *Informer) waitForCacheSync(stopCh <-chan struct{}) {
	informer.factory.WaitForCacheSync(stopCh)
	informer.releaseConifgFactory.WaitForCacheSync(stopCh)
	if informer.instanceFactory != nil {
		informer.instanceFactory.WaitForCacheSync(stopCh)
	}
	if informer.migrationFactory != nil {
		informer.migrationFactory.WaitForCacheSync(stopCh)
	}
}

func (informer *Informer) searchEvents(namespace string, objOrRef runtime.Object) (*corev1.EventList, error) {
	return informer.client.CoreV1().Events(namespace).Search(runtime.NewScheme(), objOrRef)
}

func (informer *Informer) getDependencyMetaByInstance(instance *beta1.ApplicationInstance) (*k8s.DependencyMeta, error) {
	dummyServiceSelectorStr := fmt.Sprintf("transwarp.meta=true,transwarp.install=%s", instance.Spec.InstanceId)

	dummyServices, err := informer.ListServices(instance.Namespace, dummyServiceSelectorStr)
	if err != nil {
		klog.Errorf("failed to list dummy services : %s", err.Error())
		return nil, err
	}
	if len(dummyServices) == 0 {
		return nil, nil
	}
	svc := dummyServices[0]
	metaString, found := svc.Annotations["transwarp.meta"]
	if !found {
		return nil, nil
	}

	return k8sutils.GetDependencyMetaFromDummyServiceMetaStr(metaString)
}

func NewInformer(
	client *kubernetes.Clientset,
	releaseConfigClient *releaseconfigclientset.Clientset,
	instanceClient *instanceclientset.Clientset,
	migrationClient *migrationclientset.Clientset,
	resyncPeriod time.Duration, stopCh <-chan struct{},
) (*Informer) {
	informer := &Informer{}
	informer.client = client
	informer.factory = informers.NewSharedInformerFactory(client, resyncPeriod)
	informer.deploymentLister = informer.factory.Extensions().V1beta1().Deployments().Lister()
	informer.configMapLister = informer.factory.Core().V1().ConfigMaps().Lister()
	informer.daemonSetLister = informer.factory.Extensions().V1beta1().DaemonSets().Lister()
	informer.ingressLister = informer.factory.Extensions().V1beta1().Ingresses().Lister()
	informer.jobLister = informer.factory.Batch().V1().Jobs().Lister()
	informer.podLister = informer.factory.Core().V1().Pods().Lister()
	informer.secretLister = informer.factory.Core().V1().Secrets().Lister()
	informer.serviceLister = informer.factory.Core().V1().Services().Lister()
	informer.statefulSetLister = informer.factory.Apps().V1beta1().StatefulSets().Lister()
	informer.nodeLister = informer.factory.Core().V1().Nodes().Lister()
	informer.namespaceLister = informer.factory.Core().V1().Namespaces().Lister()
	informer.resourceQuotaLister = informer.factory.Core().V1().ResourceQuotas().Lister()
	informer.persistentVolumeClaimLister = informer.factory.Core().V1().PersistentVolumeClaims().Lister()
	informer.storageClassLister = informer.factory.Storage().V1().StorageClasses().Lister()
	informer.endpointsLister = informer.factory.Core().V1().Endpoints().Lister()
	informer.limitRangeLister = informer.factory.Core().V1().LimitRanges().Lister()
	informer.replicaSetLister = informer.factory.Apps().V1().ReplicaSets().Lister()
	informer.releaseConifgFactory = releaseconfigexternalversions.NewSharedInformerFactory(releaseConfigClient, resyncPeriod)
	informer.releaseConfigLister = informer.releaseConifgFactory.Transwarp().V1beta1().ReleaseConfigs().Lister()

	if instanceClient != nil {
		informer.instanceFactory = instanceexternalversions.NewSharedInformerFactory(instanceClient, resyncPeriod)
		informer.instanceLister = informer.instanceFactory.Transwarp().V1beta1().ApplicationInstances().Lister()
	}

	if migrationClient != nil {

		informer.migrationFactory = migrationexternalversions.NewSharedInformerFactory(migrationClient, resyncPeriod)
		informer.migrationLister = informer.migrationFactory.Apiextensions().V1beta1().Migs().Lister()
	}

	informer.start(stopCh)
	informer.waitForCacheSync(stopCh)
	klog.Info("k8s cache sync finished")
	return informer
}
