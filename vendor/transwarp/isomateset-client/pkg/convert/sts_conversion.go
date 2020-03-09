/*
Copyright 2019 Transwarp All rights reserved.
*/

package v1alpha1

import (
	"encoding/json"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	transwarpv1alpha1 "transwarp/isomateset-client/pkg/apis/apiextensions.transwarp.io/v1alpha1"
	clientv1alpha1 "transwarp/isomateset-client/pkg/client/clientset/versioned/typed/apiextensions.transwarp.io/v1alpha1"
)

type GroupVersionKindName struct {
	schema.GroupVersionKind
	types.NamespacedName
}

func (gvkn *GroupVersionKindName) Defaults() {
	if gvkn.Group == "" {
		gvkn.Group = "apps"
	}
	if gvkn.Version == "" {
		gvkn.Group = "v1"
	}
	if gvkn.Kind == "" {
		gvkn.Kind = "StatefulSet"
	}
	if gvkn.Namespace == "" {
		gvkn.Namespace = "default"
	}
}

func (gvkn *GroupVersionKindName) ValidateStatefulSet() error {
	if gvkn.Group != "apps" {
		return fmt.Errorf("invalid group %s", gvkn.Group)
	}
	if !(gvkn.Version == "v1" || gvkn.Version == "v1beta1" || gvkn.Version == "v1beta2") {
		return fmt.Errorf("invalid version %s", gvkn.Version)
	}
	if gvkn.Kind != "StatefulSet" {
		return fmt.Errorf("invalid kind %s", gvkn.Kind)
	}
	return nil
}

func (gvkn *GroupVersionKindName) ValidateIsomateSet() error {
	if gvkn.Group != "apiextensions.transwarp.io" {
		return fmt.Errorf("invalid group %s", gvkn.Group)
	}
	if gvkn.Version != "v1alpha1" {
		return fmt.Errorf("invalid version %s", gvkn.Version)
	}
	if gvkn.Kind != "IsomateSet" {
		return fmt.Errorf("invalid kind %s", gvkn.Kind)
	}
	return nil
}

func convert_appsv1_StatefulSetSpec_To_v1alpha1_IsomateSetVersionTemplateSpec(in *appsv1.StatefulSetSpec, version string, out *transwarpv1alpha1.IsomateSetSpec) error {
	if out == nil {
		return fmt.Errorf("nil isomateset spec")
	}
	if out.VersionTemplates == nil {
		out.VersionTemplates = make(map[string]*transwarpv1alpha1.VersionTemplateSpec)
	}
	if _, ok := out.VersionTemplates[version]; ok {
		return fmt.Errorf("version name conflicts, cannot convert to an already existed version '%s' ", version)
	}
	vSpec := new(transwarpv1alpha1.VersionTemplateSpec)
	if in.Replicas != nil {
		vSpec.Replicas = new(int32)
		*vSpec.Replicas = *in.Replicas
	}
	if vSpec.Labels == nil {
		vSpec.Labels = make(map[string]string)
	}
	vSpec.Labels[transwarpv1alpha1.IsomateSetVersionNameLabel] = version

	// if err := k8s_api_v1.Convert_v1_PodTemplateSpec_To_core_PodTemplateSpec(&in.Template, &out.Template, s); err != nil {
	// 	return err
	// }
	podTemplateSpec := new(v1.PodTemplateSpec)
	in.Template.DeepCopyInto(podTemplateSpec)
	vSpec.Template = *podTemplateSpec

	convert_StatefulSet_VolumeClaimTemplates_To_IsomateSet_VolumeClaimTemplates(&in.VolumeClaimTemplates, &out.VolumeClaimTemplates, vSpec)

	vSpec.UpdateStrategy.Type = transwarpv1alpha1.IsomateSetUpdateStrategyType(in.UpdateStrategy.Type)
	if in.UpdateStrategy.RollingUpdate != nil {
		vSpec.UpdateStrategy.RollingUpdate = new(transwarpv1alpha1.RollingUpdateIsomateSetStrategy)
		vSpec.UpdateStrategy.RollingUpdate.Partition = new(int32)
		*vSpec.UpdateStrategy.RollingUpdate.Partition = *in.UpdateStrategy.RollingUpdate.Partition
	}

	vSpec.ServiceName = in.ServiceName
	vSpec.PodManagementPolicy = transwarpv1alpha1.PodManagementPolicyType(in.PodManagementPolicy)
	out.VersionTemplates[version] = new(transwarpv1alpha1.VersionTemplateSpec)
	vSpec.DeepCopyInto(out.VersionTemplates[version])
	return nil
}

func convert_StatefulSet_VolumeClaimTemplates_To_IsomateSet_VolumeClaimTemplates(
	in, out *[]v1.PersistentVolumeClaim,
	vSpec *transwarpv1alpha1.VersionTemplateSpec) {
	if in != nil {
		pvcs := *in
		numVol := len(pvcs)
		names := make([]string, len(pvcs))
		for i := 0; i < numVol; i++ {
			pvcs[i].Status = v1.PersistentVolumeClaimStatus{}
			names[i] = pvcs[i].GetName()
		}
		vSpec.VolumeStrategy.Names = append(vSpec.VolumeStrategy.Names, names...)
		*out = append(*out, pvcs...)
	}
}

func Convert_StatefulSets_To_v1alpha1_IsomateSet(in ...runtime.Object) (*transwarpv1alpha1.IsomateSet, error) {
	out := new(transwarpv1alpha1.IsomateSet)
	err := Convert_StatefulSets_Into_v1alpha1_IsomateSet(out, in...)
	if err != nil {
		return nil, err
	}
	return out, err
}

// merge multiple in objs into one single out obj
func Convert_StatefulSets_Into_v1alpha1_IsomateSet(
	out *transwarpv1alpha1.IsomateSet,
	in ...runtime.Object) error {
	if out == nil {
		return fmt.Errorf("out object must not be nil")
	}
	if in == nil || len(in) == 0 {
		return nil
	}

	for _, obj := range in {
		b, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		sts := new(appsv1.StatefulSet)
		if err := json.Unmarshal(b, sts); err != nil {
			return err
		}
		convert_appsv1_Statefulset_To_v1alpha1_IsomateSet(sts, out)
	}
	out.SetGroupVersionKind(transwarpv1alpha1.SchemeGroupVersion.WithKind("IsomateSet"))
	return nil
}

func convert_appsv1_StatefulSetMeta_To_v1alpha1_IsomateSetVersionTemplateMeta(in metav1.ObjectMeta, version string, out *metav1.ObjectMeta) error {
	if out == nil {
		out = new(metav1.ObjectMeta)
	}
	if out.Labels == nil {
		out.Labels = make(map[string]string)
	}
	for k, v := range in.Labels {
		out.Labels[k] = v
	}
	if out.Annotations == nil {
		out.Annotations = make(map[string]string)
	}
	for k, v := range in.Annotations {
		out.Annotations[k] = v
	}
	return nil
}
func convert_appsv1_Statefulset_To_v1alpha1_IsomateSet(in *appsv1.StatefulSet, out *transwarpv1alpha1.IsomateSet) error {
	version := in.GetName()
	if out.GetNamespace() == "" {
		out.SetNamespace(in.GetNamespace())
	}
	if out.GetName() == "" {
		out.SetName(in.GetName())
	}
	if out.Spec.Selector == nil {
		out.Spec.Selector = new(metav1.LabelSelector)
	}
	if out.Spec.Selector.MatchLabels == nil {
		out.Spec.Selector.MatchLabels = make(map[string]string)
	}
	out.Spec.Selector.MatchLabels[transwarpv1alpha1.IsomateSetNameLabel] = out.GetName()

	if err := convert_appsv1_StatefulSetSpec_To_v1alpha1_IsomateSetVersionTemplateSpec(&in.Spec, version, &out.Spec); err != nil {
		return err
	}
	if err := convert_appsv1_StatefulSetMeta_To_v1alpha1_IsomateSetVersionTemplateMeta(in.ObjectMeta, version, &out.Spec.VersionTemplates[version].ObjectMeta); err != nil {
		return err
	}
	return nil
}

func Convert_Incluster_StatefulSet_To_v1alpha1_IsomateSet(
	k8sClient kubernetes.Interface,
	imsClient clientv1alpha1.ApiextensionsV1alpha1Interface,
	out *transwarpv1alpha1.IsomateSet,
	in ...GroupVersionKindName) error {

	if out == nil {
		return fmt.Errorf("out object must not be nil")
	}
	for _, gvkn := range in {
		gvkn.Defaults()
		if gvkn.Kind != "StatefulSet" {
			klog.V(4).Infoln("not a StatefulSet resources, skip conversion")
			return nil
		}
		if gvkn.Group != "apps" {
			klog.V(4).Infof("unsupport group %s", gvkn.Group)
			return nil
		}

		switch gvkn.Version {
		case "v1":
			sts, err := k8sClient.AppsV1().StatefulSets(gvkn.Namespace).Get(gvkn.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// if sts.Annotations
			if err = Convert_StatefulSets_Into_v1alpha1_IsomateSet(out, sts); err != nil {
				return err
			}
		case "v1beta1":
			sts, err := k8sClient.AppsV1beta1().StatefulSets(gvkn.Namespace).Get(gvkn.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if err = Convert_StatefulSets_Into_v1alpha1_IsomateSet(out, sts); err != nil {
				return err
			}
		case "v1beta2":
			sts, err := k8sClient.AppsV1beta2().StatefulSets(gvkn.Namespace).Get(gvkn.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if err = Convert_StatefulSets_Into_v1alpha1_IsomateSet(out, sts); err != nil {
				return err
			}
		}
	}
	return nil

}
