package plugins

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
)

const (
	PauseReleasePluginName = "PauseRelease"
	UsePodOfflineKey       = "Transwarp_Walm_Use_Pod_Offline"
	UsePodOfflineValue     = "true"
	PodOfflineKey          = "offline-pod.transwarp.io/all-ordinals"
)

func init() {
	register(PauseReleasePluginName, &WalmPluginRunner{
		Run:  PauseRelease,
		Type: Pre_Install,
	})
}

func PauseRelease(context *PluginContext, args string) (err error) {
	newResource := []runtime.Object{}
	for _, resource := range context.Resources {
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Deployment":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Infof("failed to convert unstructured : %s", err.Error())
				return err
			}
			deployment, err := buildDeployment(converted)
			if err != nil {
				klog.Infof("failed to build deployment : %s", err.Error())
				return err
			}
			pasueDeployment(deployment)
			newResource = append(newResource, deployment)
		case "StatefulSet":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Infof("failed to convert unstructured : %s", err.Error())
				return err
			}
			statefulSet, err := buildStatefulSet(converted)
			if err != nil {
				klog.Infof("failed to build statefulSet : %s", err.Error())
				return err
			}
			pauseStatefulSet(statefulSet)
			newResource = append(newResource, statefulSet)
		default:
			newResource = append(newResource, resource)
		}
	}
	context.Resources = newResource
	return
}

func pauseStatefulSet(statefulSet *appsv1.StatefulSet) {
	if len(statefulSet.Annotations) > 0 && statefulSet.Annotations[UsePodOfflineKey] == UsePodOfflineValue {
		statefulSet.Annotations[PodOfflineKey] = ""
	} else {
		replicas := int32(0)
		statefulSet.Spec.Replicas = &replicas
	}
}

func pasueDeployment(deployment *appsv1.Deployment) {
	replicas := int32(0)
	deployment.Spec.Replicas = &replicas
}
