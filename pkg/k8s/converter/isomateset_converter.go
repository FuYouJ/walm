package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"k8s.io/api/core/v1"
	"WarpCloud/walm/pkg/k8s/utils"
	"transwarp/isomateset-client/pkg/apis/apiextensions.transwarp.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/klog"
	"fmt"
)

const (
	podVersionLabel = "isomateset.transwarp.io/pod-version"
)

func ConvertIsomateSetFromK8s(oriIsomateSet *v1alpha1.IsomateSet, pods []*v1.Pod) (walmIsomateSet *k8s.IsomateSet, err error) {
	if oriIsomateSet == nil {
		return
	}
	isomateSet := oriIsomateSet.DeepCopy()

	walmIsomateSet = &k8s.IsomateSet{
		Meta:        k8s.NewEmptyStateMeta(k8s.IsomateSetKind, isomateSet.Namespace, isomateSet.Name),
		UID:         string(isomateSet.UID),
		Labels:      isomateSet.Labels,
		Annotations: isomateSet.Annotations,
	}

	walmIsomateSet.Selector, err = utils.ConvertLabelSelectorToStr(isomateSet.Spec.Selector)
	if err != nil {
		return
	}

	versionPods, err := buildVersionPods(pods)
	if err != nil {
		klog.Errorf("failed to build version pods : %s", err.Error())
		return nil, err
	}

	for name, versionTemplateSpec := range isomateSet.Spec.VersionTemplates {
		walmVersionTemplate := &k8s.VersionTemplate{
			Name:        name,
			Labels:      versionTemplateSpec.Labels,
			Annotations: versionTemplateSpec.Annotations,
			Pods:        versionPods[name],
		}
		if versionTemplateSpec.Replicas == nil {
			walmVersionTemplate.ExpectedReplicas = 1
		} else {
			walmVersionTemplate.ExpectedReplicas = *versionTemplateSpec.Replicas
		}

		if isomateSet.Status.VersionTemplateStatus != nil {
			if versionTemplateStatus, ok := isomateSet.Status.VersionTemplateStatus[name]; ok {
				walmVersionTemplate.CurrentVersion = versionTemplateStatus.CurrentRevision
				walmVersionTemplate.UpdateVersion = versionTemplateStatus.UpdateRevision
				walmVersionTemplate.ReadyReplicas = versionTemplateStatus.ReadyReplicas
			}
		}

		walmIsomateSet.VersionTemplates = append(walmIsomateSet.VersionTemplates, walmVersionTemplate)
	}

	//for _, pod := range pods {
	//	walmPod, err := ConvertPodFromK8s(pod)
	//	if err != nil {
	//		return nil, err
	//	}
	//	walmIsomateSet.Pods = append(walmIsomateSet.Pods, walmPod)
	//}
	walmIsomateSet.State = buildWalmIsomateSetState(isomateSet, versionPods)
	return walmIsomateSet, nil
}

func buildVersionPods(pods []*v1.Pod) (map[string][]*k8s.Pod, error) {
	versionPods := map[string][]*k8s.Pod{}
	for _, pod := range pods {
		walmPod, err := ConvertPodFromK8s(pod)
		if err != nil {
			return nil, err
		}
		if len(walmPod.Labels) > 0 {
			if version, ok := walmPod.Labels[podVersionLabel]; ok {
				versionPods[version] = append(versionPods[version], walmPod)
			}
		}
	}
	return versionPods, nil
}

func buildWalmIsomateSetState(isomateSet *v1alpha1.IsomateSet, versionPods map[string][]*k8s.Pod) (walmState k8s.State) {
	if len(isomateSet.Spec.VersionTemplates) > 0 && isomateSet.Status.VersionTemplateStatus == nil {
		walmState = k8s.NewState("Pending", "IsomateSetVersionTemplateStatusNil",
			fmt.Sprintf("IsomateSet %s/%s version template status is nil, maybe IsomateSet Operator is not working now", isomateSet.Namespace, isomateSet.Name))
	} else {
		walmState = k8s.NewState("Ready", "", "")
		for versionName, versionTemplateSpec := range isomateSet.Spec.VersionTemplates {
			if !isVersionReady(versionTemplateSpec, isomateSet.Status.VersionTemplateStatus[versionName]) {
				walmState = buildWalmStateByPods(versionPods[versionName], "IsomateSet")
				break
			} else {
				isAnyPodTerminating, reason, message := isAnyPodTerminating(versionPods[versionName])
				if isAnyPodTerminating {
					walmState = k8s.NewState("Pending", reason, message)
					break
				}
			}
		}
	}

	return walmState
}

func isAnyPodTerminating(pods []*k8s.Pod) (bool, string, string) {
	for _, pod := range pods {
		if pod.State.Status == "Terminating" {
			return true, "PodTerminating", fmt.Sprintf("Pod %s/%s is in state Terminating", pod.Namespace, pod.Name)
		}
	}
	return false, "", ""
}


func isVersionReady(versionTemplateSpec *v1alpha1.VersionTemplateSpec, versionTemplateStatus *v1alpha1.TemplateStatus) bool {
	if versionTemplateStatus == nil {
		return false
	}

	if versionTemplateSpec.Replicas != nil && versionTemplateStatus.ReadyReplicas < *versionTemplateSpec.Replicas {
		return false
	}

	if versionTemplateSpec.UpdateStrategy.Type == appsv1.RollingUpdateStatefulSetStrategyType && versionTemplateSpec.UpdateStrategy.RollingUpdate != nil {
		if versionTemplateSpec.Replicas != nil && versionTemplateSpec.UpdateStrategy.RollingUpdate.Partition != nil {
			if versionTemplateStatus.UpdatedReplicas < (*versionTemplateSpec.Replicas - *versionTemplateSpec.UpdateStrategy.RollingUpdate.Partition) {
				return false
			}
			return true
		}
	}

	if versionTemplateStatus.UpdateRevision != versionTemplateStatus.CurrentRevision {
		return false
	}
	return true
}
