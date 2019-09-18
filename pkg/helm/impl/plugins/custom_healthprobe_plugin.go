package plugins

import (
	"encoding/json"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
)

const (
	CustomHealthProbePluginName = "CustomHealthProbe"
)

func init() {
	register(CustomHealthProbePluginName, &WalmPluginRunner{
		Run:  CustomHealthProbeTransform,
		Type: Pre_Install,
	})
}

type CustomProbeResource struct {
	Kind         string `json:"kind"`
	ResourceName string `json:"resourceName"`
}

type CustomHealthProbeArgs struct {
	DisableLivenessProbeResourceList  []*CustomProbeResource `json:"disableLivenessProbeResourceList"`
	DisableReadinessProbeResourceList []*CustomProbeResource `json:"disableReadinessProbeResourceList"`
	DisableAllLivenessProbe           bool                   `json:"disableAllLivenessProbe"`
	DisableAllReadinessProbe          bool                   `json:"disableAllReadinessProbe"`
}

func CustomHealthProbeTransform(context *PluginContext, args string) error {
	if args == "" {
		klog.Errorf("ignore ingress plugin, because plugin args is empty")
		return nil
	} else {
		klog.Infof("health probe pod args : %s", args)
	}

	customHealthProbeArgs := &CustomHealthProbeArgs{}
	err := json.Unmarshal([]byte(args), customHealthProbeArgs)
	if err != nil {
		klog.Errorf("failed to unmarshal plugin args : %s", err.Error())
		return err
	}

	newResource := []runtime.Object{}
	for _, resource := range context.Resources {
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Deployment":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Errorf("failed to convert unstructured : %s", err.Error())
				return err
			}
			deployment, err := buildDeployment(converted)
			if err != nil {
				klog.Errorf("buildDeployment %v error %v", converted, err)
				return err
			}
			customHealthProbeDeployment(deployment, customHealthProbeArgs)
			unstructuredObj, err := convertToUnstructured(deployment)
			if err != nil {
				klog.Infof("failed to convertToUnstructured : %v", *deployment)
				return err
			}
			newResource = append(newResource, unstructuredObj)
		case "StatefulSet":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Errorf("failed to convert unstructured : %s", err.Error())
				return err
			}
			statefulSet, err := buildStatefulSet(converted)
			if err != nil {
				klog.Errorf("buildStatefulSet %v error %v", converted, err)
				return err
			}
			customHealthProbeStatefulSet(statefulSet, customHealthProbeArgs)
			unstructuredObj, err := convertToUnstructured(statefulSet)
			if err != nil {
				klog.Infof("failed to convertToUnstructured : %v", *statefulSet)
				return err
			}
			newResource = append(newResource, unstructuredObj)
		default:
			newResource = append(newResource, resource)
		}
	}

	context.Resources = newResource
	return nil
}

func customHealthProbeStatefulSet(statefulSet *appsv1.StatefulSet, customHealthProbeArgs *CustomHealthProbeArgs) {
	disableLivenessProbe := false
	disableReadinessProbe := false

	for _, probeResource := range customHealthProbeArgs.DisableLivenessProbeResourceList {
		if probeResource.Kind == "StatefulSet" && probeResource.ResourceName == statefulSet.Name {
			disableLivenessProbe = true
		}
	}
	for _, probeResource := range customHealthProbeArgs.DisableReadinessProbeResourceList {
		if probeResource.Kind == "StatefulSet" && probeResource.ResourceName == statefulSet.Name {
			disableReadinessProbe = true
		}
	}
	if customHealthProbeArgs.DisableAllLivenessProbe {
		disableLivenessProbe = true
	}
	if customHealthProbeArgs.DisableAllReadinessProbe {
		disableReadinessProbe = true
	}

	for idx, _ := range statefulSet.Spec.Template.Spec.Containers {
		if disableLivenessProbe {
			statefulSet.Spec.Template.Spec.Containers[idx].LivenessProbe = nil
		}
		if disableReadinessProbe {
			statefulSet.Spec.Template.Spec.Containers[idx].ReadinessProbe = nil
		}
	}
}

func customHealthProbeDeployment(deployment *appsv1.Deployment, customHealthProbeArgs *CustomHealthProbeArgs) {
	disableLivenessProbe := false
	disableReadinessProbe := false

	for _, probeResource := range customHealthProbeArgs.DisableLivenessProbeResourceList {
		if probeResource.Kind == "Deployment" && probeResource.ResourceName == deployment.Name {
			disableLivenessProbe = true
		}
	}
	for _, probeResource := range customHealthProbeArgs.DisableReadinessProbeResourceList {
		if probeResource.Kind == "Deployment" && probeResource.ResourceName == deployment.Name {
			disableReadinessProbe = true
		}
	}
	if customHealthProbeArgs.DisableAllLivenessProbe {
		disableLivenessProbe = true
	}
	if customHealthProbeArgs.DisableAllReadinessProbe {
		disableReadinessProbe = true
	}

	for idx, _ := range deployment.Spec.Template.Spec.Containers {
		if disableLivenessProbe {
			klog.Infof("remove livenessProbe %v", deployment.Spec.Template.Spec.Containers[idx].LivenessProbe)
			deployment.Spec.Template.Spec.Containers[idx].LivenessProbe = nil
		}
		if disableReadinessProbe {
			klog.Infof("remove readinessProbe %v", deployment.Spec.Template.Spec.Containers[idx].ReadinessProbe)
			deployment.Spec.Template.Spec.Containers[idx].ReadinessProbe = nil
		}
	}
}
