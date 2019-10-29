package informer

import (
	"WarpCloud/walm/pkg/k8s/converter"
	"WarpCloud/walm/pkg/k8s/utils"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"transwarp/application-instance/pkg/apis/transwarp/v1beta1"
)

func (informer *Informer) getReleaseConfig(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.releaseConfigLister.ReleaseConfigs(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.ReleaseConfig{
			Meta: k8s.NewNotFoundMeta(k8s.ReleaseConfigKind, namespace, name),
		})
	}
	return converter.ConvertReleaseConfigFromK8s(resource)
}

func (informer *Informer) getConfigMap(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.configMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.ConfigMap{
			Meta: k8s.NewNotFoundMeta(k8s.ConfigMapKind, namespace, name),
		})
	}
	return converter.ConvertConfigMapFromK8s(resource)
}

func (informer *Informer) getPvc(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.persistentVolumeClaimLister.PersistentVolumeClaims(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.PersistentVolumeClaim{
			Meta: k8s.NewNotFoundMeta(k8s.PersistentVolumeClaimKind, namespace, name),
		})
	}
	return converter.ConvertPvcFromK8s(resource)
}

func (informer *Informer) getDaemonSet(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.daemonSetLister.DaemonSets(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.DaemonSet{
			Meta: k8s.NewNotFoundMeta(k8s.DaemonSetKind, namespace, name),
		})
	}
	pods, err := informer.listPods(namespace, resource.Spec.Selector, false)
	if err != nil {
		return nil, err
	}
	pods, err = informer.filterPodsByOwnerRef(k8s.DaemonSetKind, pods)
	if err != nil {
		return nil, err
	}
	return converter.ConvertDaemonSetFromK8s(resource, pods)
}

func (informer *Informer) getDeployment(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.Deployment{
			Meta: k8s.NewNotFoundMeta(k8s.DeploymentKind, namespace, name),
		})
	}
	pods, err := informer.listPods(namespace, resource.Spec.Selector, false)
	if err != nil {
		return nil, err
	}
	pods, err = informer.filterPodsByOwnerRef(k8s.DeploymentKind, pods)
	if err != nil {
		return nil, err
	}
	return converter.ConvertDeploymentFromK8s(resource, pods)
}

func (informer *Informer) getIngress(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.ingressLister.Ingresses(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.Ingress{
			Meta: k8s.NewNotFoundMeta(k8s.IngressKind, namespace, name),
		})
	}
	return converter.ConvertIngressFromK8s(resource)
}

func (informer *Informer) getJob(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.jobLister.Jobs(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.Job{
			Meta: k8s.NewNotFoundMeta(k8s.JobKind, namespace, name),
		})
	}
	pods, err := informer.listPods(namespace, resource.Spec.Selector, true)
	if err != nil {
		return nil, err
	}
	pods, err = informer.filterPodsByOwnerRef(k8s.JobKind, pods)
	if err != nil {
		return nil, err
	}
	return converter.ConvertJobFromK8s(resource, pods)
}

func (informer *Informer) getSecret(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.secretLister.Secrets(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.Secret{
			Meta: k8s.NewNotFoundMeta(k8s.SecretKind, namespace, name),
		})
	}
	return converter.ConvertSecretFromK8s(resource)
}

func (informer *Informer) getService(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.serviceLister.Services(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.Service{
			Meta: k8s.NewNotFoundMeta(k8s.ServiceKind, namespace, name),
		})
	}

	endpoints, err := informer.getEndpoints(namespace, name)
	if err != nil && !errorModel.IsNotFoundError(err) {
		return nil, err
	}
	return converter.ConvertServiceFromK8s(resource, endpoints)
}

func (informer *Informer) getStatefulSet(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.statefulSetLister.StatefulSets(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.StatefulSet{
			Meta: k8s.NewNotFoundMeta(k8s.StatefulSetKind, namespace, name),
		})
	}
	pods, err := informer.listPods(namespace, resource.Spec.Selector, false)
	if err != nil {
		return nil, err
	}
	pods, err = informer.filterPodsByOwnerRef(k8s.StatefulSetKind, pods)
	if err != nil {

		return nil, err
	}
	return converter.ConvertStatefulSetFromK8s(resource, pods)
}

func (informer *Informer) filterPodsByOwnerRef(resKind k8s.ResourceKind, pods []*corev1.Pod) ([]*corev1.Pod, error) {

	var podList []*corev1.Pod

	switch resKind {
	case k8s.DeploymentKind:
		for _, pod := range pods {
			for _, ownerRef := range pod.OwnerReferences {

				isFind := false
				if ownerRef.Kind != string(k8s.ReplicaSetKind) {
					continue
				}
				replicaSet, err := informer.replicaSetLister.ReplicaSets(pod.Namespace).Get(ownerRef.Name)
				if err != nil {
					return nil, err
				}

				if ownerRef.UID != replicaSet.UID {
					continue
				}
				for _, rsOwnerRef := range replicaSet.OwnerReferences {
					if rsOwnerRef.Kind == string(k8s.DeploymentKind) {
						deployment, err := informer.deploymentLister.Deployments(replicaSet.Namespace).Get(rsOwnerRef.Name)
						if err != nil {
							return nil, err
						}
						if rsOwnerRef.UID != deployment.UID {
							break
						}
						isFind = true
						break
					}
				}
				if isFind {
					podList = append(podList, pod)
					break
				}
			}
		}
	case k8s.StatefulSetKind, k8s.DaemonSetKind, k8s.JobKind:
		for _, pod := range pods {
			for _, ownerRef := range pod.OwnerReferences {
				if string(resKind) == ownerRef.Kind {

					if resKind == k8s.StatefulSetKind {
						resource, err := informer.statefulSetLister.StatefulSets(pod.Namespace).Get(ownerRef.Name)
						if err != nil {
							return nil, err
						}
						if resource.UID != ownerRef.UID {
							continue
						}

					} else if resKind == k8s.DaemonSetKind {
						resource, err := informer.daemonSetLister.DaemonSets(pod.Namespace).Get(ownerRef.Name)
						if err != nil {
							return nil, err
						}
						if resource.UID != ownerRef.UID {
							continue
						}
					} else {
						resource, err := informer.jobLister.Jobs(pod.Namespace).Get(ownerRef.Name)
						if err != nil {
							return nil, err
						}
						if resource.UID != ownerRef.UID {
							continue
						}
					}

					podList = append(podList, pod)
					break
				}
			}
		}
	default:

	}
	return pods, nil
}

func (informer *Informer) listPods(namespace string, labelSelector *metav1.LabelSelector, fromJob bool) ([]*corev1.Pod, error) {
	selector, err := utils.ConvertLabelSelectorToSelector(labelSelector)
	if err != nil {
		klog.Errorf("failed to convert label selector : %s", err.Error())
		return nil, err
	}
	pods, err := informer.podLister.Pods(namespace).List(selector)
	var walmPods []*corev1.Pod
	for _, pod := range pods {
		if !fromJob {
			if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
				continue
			}
		}
		walmPods = append(walmPods, pod)
	}
	if err != nil {
		klog.Errorf("failed to list pods : %s", err.Error())
		return nil, err
	}
	return walmPods, nil
}

func (informer *Informer) getEndpoints(namespace, name string) (*corev1.Endpoints, error) {
	endpoints, err := informer.endpointsLister.Endpoints(namespace).Get(name)
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("endpoints %s/%s is not found", namespace, name)
			return nil, errorModel.NotFoundError{}
		}
		klog.Errorf("failed to get endpoints : %s", err.Error())
		return nil, err
	}

	return endpoints, nil
}

func (informer *Informer) getNode(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.nodeLister.Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.Node{
			Meta: k8s.NewNotFoundMeta(k8s.NodeKind, namespace, name),
		})
	}

	podsOnNode, err := informer.getNonTermiatedPodsOnNode(name, nil)
	if err != nil {
		klog.Errorf("failed to get pods on node : %s", err.Error())
		return nil, err
	}
	return converter.ConvertNodeFromK8s(resource, podsOnNode)
}

func (informer *Informer) getNonTermiatedPodsOnNode(nodeName string, labelSelector *metav1.LabelSelector) (*corev1.PodList, error) {
	selector, err := utils.ConvertLabelSelectorToSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	pods, err := informer.podLister.Pods("").List(selector)
	if err != nil {
		klog.Errorf("failed to list pods : %s", err.Error())
		return nil, err
	}

	podList := &corev1.PodList{
		Items: []corev1.Pod{},
	}

	for _, pod := range pods {
		if pod.Spec.NodeName == nodeName && pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			podList.Items = append(podList.Items, *pod)
		}
	}

	return podList, nil

}

func (informer *Informer) getStorageClass(namespace, name string) (k8s.Resource, error) {
	resource, err := informer.storageClassLister.Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.StorageClass{
			Meta: k8s.NewNotFoundMeta(k8s.StorageClassKind, namespace, name),
		})
	}

	return converter.ConvertStorageClassFromK8s(resource)
}

func (informer *Informer) getInstance(namespace, name string) (k8s.Resource, error) {
	notFoundResource := &k8s.ApplicationInstance{
		Meta: k8s.NewNotFoundMeta(k8s.InstanceKind, namespace, name),
	}
	if informer.instanceLister == nil {
		return notFoundResource, errorModel.NotFoundError{}
	}
	resource, err := informer.instanceLister.ApplicationInstances(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, notFoundResource)
	}

	resourceMetas := convertInstanceModulesToResourceMetas(resource.Status.Modules)
	instanceModules, err := informer.GetResourceSet(resourceMetas)
	if err != nil {
		klog.Errorf("failed to get instance modules : %s", err.Error())
		return nil, err
	}

	dependencyMeta, err := informer.getDependencyMetaByInstance(resource)
	if err != nil {
		klog.Errorf("failed to get dependency meta by instance : %s", err.Error())
		return nil, err
	}

	return converter.ConvertInstanceFromK8s(resource, instanceModules, dependencyMeta)
}

func (informer *Informer) getReplicaSet(namespace, name string) (k8s.Resource, error) {
	notFoundResource := &k8s.ReplicaSet{
		Meta: k8s.NewNotFoundMeta(k8s.ReplicaSetKind, namespace, name),
	}
	if informer.replicaSetLister == nil {
		return notFoundResource, errorModel.NotFoundError{}
	}

	k8sReplicaSet, err := informer.replicaSetLister.ReplicaSets(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.ReplicaSet{
			Meta: k8s.NewNotFoundMeta(k8s.ReplicaSetKind, namespace, name),
		})
	}

	return converter.ConvertReplicaSetFromK8s(k8sReplicaSet)
}

func (informer *Informer) getMigration(namespace string, name string) (k8s.Resource, error) {
	notFoundResource := &k8s.Mig{
		Meta: k8s.NewNotFoundMeta(k8s.MigKind, namespace, name),
	}
	if informer.migrationLister == nil {
		return notFoundResource, errorModel.NotFoundError{}
	}

	k8sMig, err := informer.migrationLister.Migs(namespace).Get(name)
	if err != nil {
		return convertResourceError(err, &k8s.Mig{
			Meta: k8s.NewNotFoundMeta(k8s.MigKind, namespace, name),
		})
	}

	return converter.ConvertMigFromK8s(k8sMig)
}

func convertInstanceModulesToResourceMetas(references []v1beta1.ResourceReference) []release.ReleaseResourceMeta {
	res := []release.ReleaseResourceMeta{}
	for _, ref := range references {
		res = append(res, release.ReleaseResourceMeta{
			Name:      ref.ResourceRef.Name,
			Namespace: ref.ResourceRef.Namespace,
			Kind:      k8s.ResourceKind(ref.ResourceRef.Kind),
		})
	}
	return res
}

func convertResourceError(err error, notFoundResource k8s.Resource) (k8s.Resource, error) {
	if utils.IsK8sResourceNotFoundErr(err) {
		klog.Warningf(" %s %s/%s is not found", notFoundResource.GetKind(), notFoundResource.GetNamespace(), notFoundResource.GetName())
		return notFoundResource, errorModel.NotFoundError{}
	}
	klog.Errorf("failed to get %s %s/%s : %s", notFoundResource.GetKind(), notFoundResource.GetNamespace(), notFoundResource.GetName(), err.Error())
	return nil, err
}
