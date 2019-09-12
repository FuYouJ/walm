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

func CustomHealthProbeTransform(context *PluginContext, args string) (err error) {
	if args == "" {
		klog.Errorf("ignore ingress plugin, because plugin args is empty")
		return nil
	} else {
		klog.Infof("health probe pod args : %s", args)
	}

	customHealthProbeArgs := &CustomHealthProbeArgs{}
	err = json.Unmarshal([]byte(args), customHealthProbeArgs)
	if err != nil {
		klog.Infof("failed to unmarshal plugin args : %s", err.Error())
		return err
	}

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
				klog.Errorf("buildDeployment %v error %v", converted, err)
				return err
			}
			customHealthProbeDeployment(deployment, customHealthProbeArgs)
			newResource = append(newResource, deployment)
		case "StatefulSet":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Infof("failed to convert unstructured : %s", err.Error())
				return err
			}
			statefulSet, err := buildStatefulSet(converted)
			if err != nil {
				klog.Errorf("buildStatefulSet %v error %v", converted, err)
				return err
			}
			customHealthProbeStatefulSet(statefulSet, customHealthProbeArgs)
			newResource = append(newResource, statefulSet)
		default:
			newResource = append(newResource, resource)
		}
	}

	return
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

	for _, container := range statefulSet.Spec.Template.Spec.Containers {
		if disableLivenessProbe {
			container.LivenessProbe = nil
		}
		if disableReadinessProbe {
			container.ReadinessProbe = nil
		}
	}
}

func customHealthProbeDeployment(deployment *appsv1.Deployment, customHealthProbeArgs *CustomHealthProbeArgs) {
	disableLivenessProbe := false
	disableReadinessProbe := false

	for _, probeResource := range customHealthProbeArgs.DisableLivenessProbeResourceList {
		if probeResource.Kind == "StatefulSet" && probeResource.ResourceName == deployment.Name {
			disableLivenessProbe = true
		}
	}
	for _, probeResource := range customHealthProbeArgs.DisableReadinessProbeResourceList {
		if probeResource.Kind == "StatefulSet" && probeResource.ResourceName == deployment.Name {
			disableReadinessProbe = true
		}
	}
	if customHealthProbeArgs.DisableAllLivenessProbe {
		disableLivenessProbe = true
	}
	if customHealthProbeArgs.DisableAllReadinessProbe {
		disableReadinessProbe = true
	}

	for _, container := range deployment.Spec.Template.Spec.Containers {
		if disableLivenessProbe {
			container.LivenessProbe = nil
		}
		if disableReadinessProbe {
			container.ReadinessProbe = nil
		}
	}
}
