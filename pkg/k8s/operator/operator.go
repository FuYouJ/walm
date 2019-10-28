package operator

import (
	"WarpCloud/walm/pkg/k8s"
	"WarpCloud/walm/pkg/k8s/client/helm"
	"WarpCloud/walm/pkg/k8s/converter"
	"WarpCloud/walm/pkg/k8s/utils"
	errorModel "WarpCloud/walm/pkg/models/error"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	migrationclientset "github.com/migration/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"reflect"
)

const (
	storageClassAnnotationKey = "volume.beta.kubernetes.io/storage-class"
)

type Operator struct {
	client      *kubernetes.Clientset
	k8sCache    k8s.Cache
	kubeClients *helm.Client
	k8sMigrationClient *migrationclientset.Clientset
}

func (op *Operator) DeleteStatefulSetPvcs(statefulSets []*k8sModel.StatefulSet) error {
	for _, statefulSet := range statefulSets {
		pvcs, err := op.k8sCache.ListPersistentVolumeClaims(statefulSet.Namespace, statefulSet.Selector)
		if err != nil {
			klog.Errorf("failed to list pvcs : %s", err.Error())
			return err
		}
		for _, pvc := range pvcs {
			err := op.doDeletePvc(pvc, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (op *Operator) DeletePod(namespace string, name string) error {
	err := op.client.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("pod %s/%s is not found ", namespace, name)
			return nil
		}
		klog.Errorf("failed to delete pod %s/%s : %s", namespace, name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) RestartPod(namespace string, name string) error {
	err := op.client.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("failed to restart pod %s/%s : %s", namespace, name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) MigratePod(namespace string, name string, mig *k8sModel.Mig, fromNode bool) error {


	if mig.Name == "" {
		return errors.Errorf("name for Mig must be set")
	}

	if mig.Namespace == "" {
		klog.Warningf("mig namespace is empty, set as %s default", namespace)
		mig.Namespace = namespace
	}

	mig.Spec.Namespace = namespace
	mig.Spec.PodName = name


	k8sMig, err := converter.ConvertMigToK8s(mig)
	if err != nil {
		return errors.Errorf("failed to convert mig to k8sMigration: %s", err.Error())
	}

	if fromNode {
		k8sMig.Labels = map[string]string{"migType":"node", "migName": mig.Name}
		k8sMig.Name = mig.Name + "-" + mig.Spec.Namespace + "-" + mig.Spec.PodName
	}

	_, err = op.k8sMigrationClient.ApiextensionsV1beta1().Migs(mig.Namespace).Create(k8sMig)
	if err != nil {
		return errors.Errorf("failed to migrate pod %s/%s: %s", namespace, name, err.Error())
	}
	return nil
}

func (op *Operator) MigrateNode(mig *k8sModel.Mig) error {

	k8sMigrationClient := op.k8sMigrationClient
	if k8sMigrationClient == nil {
		return errors.Errorf("failed to get migration client, check config.CrdConfig.EnableMigrationCRD")
	}

	nodeList, err := op.k8sCache.GetNodes("")
	if err != nil {
		klog.Errorf("failed to get node %s: %s", mig.SrcHost, err.Error())
		return err
	}

	findNode := false
	for _, node := range nodeList {
		if mig.SrcHost == node.Name {
			if node.UnSchedulable {
				return errors.Errorf("node is unschedulable, please wait")
			}
			findNode = true
			break
		}
	}
	if !findNode {
		return errors.Errorf("node %s not exist.", mig.SrcHost)
	}

	statefulsets, err := op.k8sCache.ListStatefulSets("", "")
	if err != nil {
		klog.Errorf("failed to get sts: %s", err.Error())
		return err
	}

	var podList []*k8sModel.Pod
	for _, sts := range statefulsets {
		for _, pod := range sts.Pods {
			if pod.NodeName == mig.SrcHost {
				podList = append(podList, pod)
			}
		}
	}

	for _, pod := range podList {
		err = op.MigratePod(pod.Namespace, pod.Name, mig, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (op *Operator) BuildManifestObjects(namespace string, manifest string) ([]map[string]interface{}, error) {
	_, kubeClient := op.kubeClients.GetKubeClient(namespace)
	resources, err := kubeClient.Build(bytes.NewBufferString(manifest))
	if err != nil {
		klog.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}

	results := []map[string]interface{}{}
	for _, resource := range resources {
		results = append(results, resource.Object.(*unstructured.Unstructured).Object)
	}
	return results, nil
}

func (op *Operator) ComputeReleaseResourcesByManifest(namespace string, manifest string) (*release.ReleaseResources, error) {
	_, kubeClient := op.kubeClients.GetKubeClient(namespace)
	resources, err := kubeClient.Build(bytes.NewBufferString(manifest))
	if err != nil {
		klog.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}

	result := &release.ReleaseResources{}
	for _, resource := range resources {
		unstructured := resource.Object.(*unstructured.Unstructured)
		switch unstructured.GetKind() {
		case "Deployment":
			releaseResourceDeployment, err := buildReleaseResourceDeployment(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource deployment %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Deployments = append(result.Deployments, releaseResourceDeployment)
		case "StatefulSet":
			releaseResourceStatefulSet, err := buildReleaseResourceStatefulSet(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource stateful set %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.StatefulSets = append(result.StatefulSets, releaseResourceStatefulSet)
		case "DaemonSet":
			releaseResourceDaemonSet, err := buildReleaseResourceDaemonSet(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource daemon set %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.DaemonSets = append(result.DaemonSets, releaseResourceDaemonSet)
		case "Job":
			releaseResourceJob, err := buildReleaseResourceJob(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource job %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Jobs = append(result.Jobs, releaseResourceJob)
		case "PersistentVolumeClaim":
			pvc, err := buildReleaseResourcePvc(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource pvc %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Pvcs = append(result.Pvcs, pvc)
		default:
		}
	}
	return result, nil
}

func buildReleaseResourceDeployment(resource *unstructured.Unstructured) (*release.ReleaseResourceDeployment, error) {
	deployment := &v1beta1.Deployment{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal deployment %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, deployment)
	if err != nil {
		klog.Errorf("failed to unmarshal deployment %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResourceDeployment := &release.ReleaseResourceDeployment{}
	if deployment.Spec.Replicas != nil {
		releaseResourceDeployment.Replicas = *deployment.Spec.Replicas
	}

	releaseResourceDeployment.ReleaseResourceBase, err = buildReleaseResourceBase(resource, deployment.Spec.Template, nil)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResourceDeployment, nil
}

func buildReleaseResourceStatefulSet(resource *unstructured.Unstructured) (*release.ReleaseResourceStatefulSet, error) {
	statefulSet := &appsv1beta1.StatefulSet{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal statefulSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, statefulSet)
	if err != nil {
		klog.Errorf("failed to unmarshal statefulSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceStatefulSet{}
	if statefulSet.Spec.Replicas != nil {
		releaseResource.Replicas = *statefulSet.Spec.Replicas
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, statefulSet.Spec.Template, statefulSet.Spec.VolumeClaimTemplates)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourceDaemonSet(resource *unstructured.Unstructured) (*release.ReleaseResourceDaemonSet, error) {
	daemonSet := &extv1beta1.DaemonSet{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal daemonSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, daemonSet)
	if err != nil {
		klog.Errorf("failed to unmarshal daemonSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceDaemonSet{
		NodeSelector: daemonSet.Spec.Template.Spec.NodeSelector,
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, daemonSet.Spec.Template, nil)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourceJob(resource *unstructured.Unstructured) (*release.ReleaseResourceJob, error) {
	job := &batchv1.Job{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal job %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, job)
	if err != nil {
		klog.Errorf("failed to unmarshal job %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceJob{}
	if job.Spec.Parallelism != nil {
		releaseResource.Parallelism = *job.Spec.Parallelism
	}
	if job.Spec.Completions != nil {
		releaseResource.Completions = *job.Spec.Completions
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, job.Spec.Template, nil)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourcePvc(resource *unstructured.Unstructured) (*release.ReleaseResourceStorage, error) {
	pvc := &v1.PersistentVolumeClaim{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal pvc %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, pvc)
	if err != nil {
		klog.Errorf("failed to unmarshal pvc %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	return buildPvcStorage(*pvc), nil
}

func buildReleaseResourceBase(r *unstructured.Unstructured, podTemplateSpec v1.PodTemplateSpec, pvcs []v1.PersistentVolumeClaim) (releaseResource release.ReleaseResourceBase, err error) {
	releaseResource = release.ReleaseResourceBase{
		Name:        r.GetName(),
		PodRequests: &release.ReleaseResourcePod{},
		PodLimits:   &release.ReleaseResourcePod{},
	}

	podRequests, podLimits := utils.GetPodRequestsAndLimits(podTemplateSpec.Spec)
	if quantity, ok := podRequests[v1.ResourceCPU]; ok {
		releaseResource.PodRequests.Cpu = float64(quantity.MilliValue()) / utils.K8sResourceCpuScale
	}
	if quantity, ok := podRequests[v1.ResourceMemory]; ok {
		releaseResource.PodRequests.Memory = quantity.Value() / utils.K8sResourceMemoryScale
	}
	if quantity, ok := podLimits[v1.ResourceCPU]; ok {
		releaseResource.PodLimits.Cpu = float64(quantity.MilliValue()) / utils.K8sResourceCpuScale
	}
	if quantity, ok := podLimits[v1.ResourceMemory]; ok {
		releaseResource.PodLimits.Memory = quantity.Value() / utils.K8sResourceMemoryScale
	}

	releaseResource.PodRequests.Storage = buildTosDiskStorage(r.Object)
	releaseResource.PodRequests.Storage = append(releaseResource.PodRequests.Storage, buildPvcStorages(pvcs)...)
	return
}

func buildTosDiskStorage(object map[string]interface{}) (tosDiskStorages []*release.ReleaseResourceStorage) {
	tosDiskStorages = []*release.ReleaseResourceStorage{}
	type TosDiskVolumeSource struct {
		Name        string        `json:"name" description:"tos disk name"`
		StorageType string        `json:"storageType" description:"tos disk storageType"`
		Capability  v1.Capability `json:"capability" description:"tos disk capability"`
	}

	volumes, found, err := unstructured.NestedSlice(object, "spec", "template", "spec", "volumes")
	if !found || err != nil {
		klog.Warning("failed to find pod volumes")
		return
	}

	for _, volume := range volumes {
		if volumeMap, ok := volume.(map[string]interface{}); ok {
			if tosDisk, ok1 := volumeMap["tosDisk"]; ok1 {
				tosDiskBytes, err := json.Marshal(tosDisk)
				if err != nil {
					klog.Warningf("failed to marshal tosDisk : %s", err.Error())
					continue
				}
				tosDiskVolumeSource := &TosDiskVolumeSource{}
				err = json.Unmarshal(tosDiskBytes, tosDiskVolumeSource)
				if err != nil {
					klog.Warningf("failed to unmarshal tosDisk : %s", err.Error())
					continue
				}

				quantity, err := resource.ParseQuantity(string(tosDiskVolumeSource.Capability))
				if err != nil {
					klog.Warningf("failed to parse quantity: %s", err.Error())
					continue
				}

				tosDiskStorages = append(tosDiskStorages, &release.ReleaseResourceStorage{
					Name:         tosDiskVolumeSource.Name,
					Type:         release.TosDiskPodStorageType,
					Size:         quantity.Value() / utils.K8sResourceStorageScale,
					StorageClass: tosDiskVolumeSource.StorageType,
				})
			}
		}
	}
	return
}

func buildPvcStorages(pvcs []v1.PersistentVolumeClaim) (pvcStorages []*release.ReleaseResourceStorage) {
	pvcStorages = []*release.ReleaseResourceStorage{}
	for _, pvc := range pvcs {
		pvcStorages = append(pvcStorages, buildPvcStorage(pvc))
	}
	return
}

func buildPvcStorage(pvc v1.PersistentVolumeClaim) *release.ReleaseResourceStorage {
	pvcStorage := &release.ReleaseResourceStorage{
		Name: pvc.Name,
		Type: release.PvcPodStorageType,
	}
	quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	pvcStorage.Size = quantity.Value() / utils.K8sResourceStorageScale
	if pvc.Spec.StorageClassName != nil {
		pvcStorage.StorageClass = *pvc.Spec.StorageClassName
	} else if len(pvc.Annotations) > 0 {
		pvcStorage.StorageClass = pvc.Annotations[storageClassAnnotationKey]
	}
	return pvcStorage
}

func (op *Operator) CreateNamespace(namespace *k8sModel.Namespace) error {
	k8sNamespace, err := converter.ConvertNamespaceToK8s(namespace)
	if err != nil {
		klog.Errorf("failed to convert namespace : %s", err.Error())
		return err
	}
	_, err = op.client.CoreV1().Namespaces().Create(k8sNamespace)
	if err != nil {
		klog.Errorf("failed to create namespace %s : %s", k8sNamespace.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) UpdateNamespace(namespace *k8sModel.Namespace) (error) {
	k8sNamespace, err := converter.ConvertNamespaceToK8s(namespace)
	if err != nil {
		klog.Errorf("failed to convert namespace : %s", err.Error())
		return err
	}
	_, err = op.client.CoreV1().Namespaces().Update(k8sNamespace)
	if err != nil {
		klog.Errorf("failed to update namespace %s : %s", k8sNamespace.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) DeleteNamespace(name string) error {
	err := op.client.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("namespace %s is not found ", name)
			return nil
		}
		klog.Errorf("failed to delete namespace %s : %s", name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) CreateResourceQuota(resourceQuota *k8sModel.ResourceQuota) error {
	k8sQuota, err := converter.ConvertResourceQuotaToK8s(resourceQuota)
	if err != nil {
		klog.Errorf("failed to convert resource quota : %s", err.Error())
		return err
	}
	_, err = op.client.CoreV1().ResourceQuotas(k8sQuota.Namespace).Create(k8sQuota)
	if err != nil {
		klog.Errorf("failed to create resource quota %s/%s : %s", k8sQuota.Namespace, k8sQuota.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) CreateOrUpdateResourceQuota(resourceQuota *k8sModel.ResourceQuota) error {
	update := true
	_, err := op.client.CoreV1().ResourceQuotas(resourceQuota.Namespace).Get(resourceQuota.Name, metav1.GetOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			update = false
		} else {
			klog.Errorf("failed to get resource quota %s/%s : %s", resourceQuota.Namespace, resourceQuota.Name, err.Error())
			return err
		}
	}

	k8sQuota, err := converter.ConvertResourceQuotaToK8s(resourceQuota)
	if err != nil {
		klog.Errorf("failed to convert resource quota : %s", err.Error())
		return err
	}

	if update {
		_, err = op.client.CoreV1().ResourceQuotas(k8sQuota.Namespace).Update(k8sQuota)
		if err != nil {
			klog.Errorf("failed to update resource quota %s/%s : %s", k8sQuota.Namespace, k8sQuota.Name, err.Error())
			return err
		}
	} else {
		_, err = op.client.CoreV1().ResourceQuotas(k8sQuota.Namespace).Create(k8sQuota)
		if err != nil {
			klog.Errorf("failed to create resource quota %s/%s : %s", k8sQuota.Namespace, k8sQuota.Name, err.Error())
			return err
		}
	}
	return nil
}

func (op *Operator) CreateLimitRange(limitRange *k8sModel.LimitRange) error {
	k8sLimitRange, err := converter.ConvertLimitRangeToK8s(limitRange)
	if err != nil {
		klog.Errorf("failed to convert limit range : %s", err.Error())
		return err
	}

	_, err = op.client.CoreV1().LimitRanges(k8sLimitRange.Namespace).Create(k8sLimitRange)
	if err != nil {
		klog.Errorf("failed to create limit range %s/%s : %s", k8sLimitRange.Namespace, k8sLimitRange.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) LabelNode(name string, labelsToAdd map[string]string, labelsToRemove []string) (err error) {
	if len(labelsToAdd) == 0 && len(labelsToRemove) == 0 {
		return
	}

	node, err := op.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	oldLabels, err := json.Marshal(node.Labels)
	if err != nil {
		return
	}

	node.Labels = utils.MergeLabels(node.Labels, labelsToAdd, labelsToRemove)
	newLabels, err := json.Marshal(node.Labels)
	if err != nil {
		return
	}

	if !reflect.DeepEqual(oldLabels, newLabels) {
		_, err = op.client.CoreV1().Nodes().Update(node)
		if err != nil {
			klog.Errorf("failed to update node %s : %s", name, err.Error())
			return
		}
	}

	return
}

func (op *Operator) AnnotateNode(name string, annotationsToAdd map[string]string, annotationsToRemove []string) (err error) {
	if len(annotationsToAdd) == 0 && len(annotationsToRemove) == 0 {
		return
	}

	node, err := op.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	oldAnnos, err := json.Marshal(node.Annotations)
	if err != nil {
		return
	}

	node.Annotations = utils.MergeLabels(node.Annotations, annotationsToAdd, annotationsToRemove)
	newAnnos, err := json.Marshal(node.Annotations)
	if err != nil {
		return
	}

	if !reflect.DeepEqual(oldAnnos, newAnnos) {
		_, err = op.client.CoreV1().Nodes().Update(node)
		if err != nil {
			klog.Errorf("failed to update node %s : %s", name, err.Error())
			return
		}
	}

	return
}

func (op *Operator) TaintNoExecuteNode(name string, taintsToAdd map[string]string, taintsToRemove []string) (err error) {
	taints := make([]v1.Taint, 0)
	noExecuteTaints := make([]v1.Taint, 0)
	if len(taintsToAdd) == 0 && len(taintsToRemove) == 0 {
		return
	}

	node, err := op.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	for _, nodeTaint := range node.Spec.Taints {
		if nodeTaint.Effect == v1.TaintEffectNoExecute {
			noExecuteTaints = append(noExecuteTaints, nodeTaint)
		} else {
			taints = append(taints, nodeTaint)
		}
	}
	for key, value := range taintsToAdd {
		found := false
		for _, noExecuteTaint := range noExecuteTaints {
			if noExecuteTaint.Key == key {
				found = true
				break
			}
		}
		if !found {
			noExecuteTaints = append(noExecuteTaints, v1.Taint{
				Key:    key,
				Value:  value,
				Effect: v1.TaintEffectNoExecute,
			})
		}
	}
	for _, key := range taintsToRemove {
		for idx, noExecuteTaint := range noExecuteTaints {
			if noExecuteTaint.Key == key {
				noExecuteTaints = append(noExecuteTaints[:idx], noExecuteTaints[idx+1:]...)
			}
		}
	}

	_, err = op.client.CoreV1().Nodes().Update(node)
	if err != nil {
		klog.Errorf("failed to update node %s : %s", name, err.Error())
		return
	}

	return
}

func (op *Operator) DeletePvc(namespace string, name string) error {
	resource, err := op.k8sCache.GetResource(k8sModel.PersistentVolumeClaimKind, namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("pvc %s/%s is not found", namespace, name)
			return nil
		}
		klog.Errorf("failed to get pvc %s/%s : %s", namespace, name, err.Error())
		return err
	}

	return op.doDeletePvc(resource.(*k8sModel.PersistentVolumeClaim), false)
}

func (op *Operator) doDeletePvc(pvc *k8sModel.PersistentVolumeClaim, force bool) error {
	if !force && len(pvc.Labels) > 0 {
		selector := &metav1.LabelSelector{
			MatchLabels: pvc.Labels,
		}

		selectorStr, err := utils.ConvertLabelSelectorToStr(selector)
		if err != nil {
			klog.Errorf("failed to convert label selector: %s", err.Error())
			return err
		}

		statefulSets, err := op.k8sCache.ListStatefulSets(pvc.Namespace, selectorStr)
		if err != nil {
			klog.Errorf("failed to list stateful set : %s", err.Error())
			return err
		}
		if len(statefulSets) > 0 {
			statefulSetNames := make([]string, len(statefulSets))
			for _, statefulSet := range statefulSets {
				statefulSetNames = append(statefulSetNames, statefulSet.Namespace+"/"+statefulSet.Name)
			}
			err = fmt.Errorf("pvc %s/%s can not be deleted, it is still used by statefulsets %v", pvc.Namespace, pvc.Name, statefulSetNames)
			return err
		}
	}
	err := op.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("pvc %s/%s is not found ", pvc.Namespace, pvc.Name)
			return nil
		}
		klog.Errorf("failed to delete pvc %s/%s : %s", pvc.Namespace, pvc.Name, err.Error())
		return err
	}
	klog.Infof("succeed to delete pvc %s/%s", pvc.Namespace, pvc.Name)
	return nil
}

func (op *Operator) DeletePvcs(namespace string, labelSeletorStr string) error {
	pvcs, err := op.k8sCache.ListPersistentVolumeClaims(namespace, labelSeletorStr)
	if err != nil {
		klog.Errorf("failed to list pvcs : %s", err.Error())
		return err
	}
	for _, pvc := range pvcs {
		err := op.doDeletePvc(pvc, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (op *Operator) CreateSecret(namespace string, secretRequestBody *k8sModel.CreateSecretRequestBody) error {
	secret, err := buildSecret(namespace, secretRequestBody)
	if err != nil {
		return err
	}
	_, err = op.client.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		klog.Errorf("failed to create secret %s/%s : %s", namespace, secretRequestBody.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) UpdateSecret(namespace string, walmSecret *k8sModel.CreateSecretRequestBody) (err error) {
	newSecret, err := buildSecret(namespace, walmSecret)
	if err != nil {
		return err
	}
	_, err = op.client.CoreV1().Secrets(namespace).Update(newSecret)
	if err != nil {
		klog.Errorf("failed to update secret : %s", err.Error())
		return
	}
	klog.Infof("succeed to update secret %s/%s", namespace, walmSecret.Name)
	return
}

func (op *Operator) DeleteSecret(namespace, name string) (err error) {
	err = op.client.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("secret %s/%s is not found ", namespace, name)
			return nil
		}
		klog.Errorf("failed to delete secret : %s", err.Error())
		return
	}
	klog.Infof("succeed to delete secret %s/%s", namespace, name)
	return
}

func buildSecret(namespace string, walmSecret *k8sModel.CreateSecretRequestBody) (secret *v1.Secret, err error) {
	DataByte := make(map[string][]byte, 0)
	for k, v := range walmSecret.Data {
		DataByte[k], err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			klog.Errorf("failed to decode secret : %+v %s", walmSecret.Data, err.Error())
			return
		}
	}
	klog.Infof("secret data: %+v", walmSecret.Data)
	secret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      walmSecret.Name,
		},
		Data: DataByte,
		Type: v1.SecretType(walmSecret.Type),
	}
	return
}

func (op *Operator) UpdateIngress(namespace, ingressName string, requestBody *k8sModel.IngressRequestBody) (err error) {
	k8sIngress, err := op.client.ExtensionsV1beta1().Ingresses(namespace).Get(ingressName, metav1.GetOptions{})
	if err != nil {
		return
	}
	if len(requestBody.Annotations) != 0 {
		k8sIngress.Annotations = requestBody.Annotations
	}
	if len(k8sIngress.Spec.Rules) > 0 {
		rule := k8sIngress.Spec.Rules[0]
		if requestBody.Host != "" {
			k8sIngress.Spec.Rules[0].Host = requestBody.Host
		}
		if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 {
			if requestBody.Path != "" {
				k8sIngress.Spec.Rules[0].HTTP.Paths[0].Path = requestBody.Path
			}
		}
	}
	if len(requestBody.Annotations) != 0 {
		k8sIngress.Annotations = requestBody.Annotations
	}
	_, err = op.client.ExtensionsV1beta1().Ingresses(namespace).Update(k8sIngress)
	if err != nil {
		klog.Errorf("failed to update ingress %s : %s", ingressName, err.Error())
		return
	}

	return
}

func (op *Operator) UpdateConfigMap(namespace, configMapName string, requestBody *k8sModel.ConfigMapRequestBody) (err error) {
	k8sConfigMap, err := op.client.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return
	}
	for key, value := range requestBody.Data {
		k8sConfigMap.Data[key] = value
	}
	_, err = op.client.CoreV1().ConfigMaps(namespace).Update(k8sConfigMap)
	if err != nil {
		klog.Errorf("failed to update configMap %s : %s", configMapName, err.Error())
		return
	}

	return
}

func NewOperator(client *kubernetes.Clientset, k8sCache k8s.Cache, kubeClients *helm.Client, k8sMigrationClient *migrationclientset.Clientset) *Operator {
	return &Operator{
		client:      client,
		k8sCache:    k8sCache,
		kubeClients: kubeClients,
		k8sMigrationClient: k8sMigrationClient,
	}
}
