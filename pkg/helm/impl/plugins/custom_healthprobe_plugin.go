package plugins

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	for _, resource := range context.Resources {
		unstructuredObj := resource.(*unstructured.Unstructured)
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Deployment", "StatefulSet":
			err := customHealthProbe(unstructuredObj, customHealthProbeArgs)
			if err != nil {
				klog.Errorf("failed to customize health probe : %s", err.Error())
				return err
			}
		}
	}

	return nil
}

func customHealthProbe(unstructuredObj *unstructured.Unstructured, customHealthProbeArgs *CustomHealthProbeArgs) error{
	disableLivenessProbe, disableReadinessProbe := buildProbeStatus(unstructuredObj, customHealthProbeArgs)

	if !(disableReadinessProbe || disableLivenessProbe) {
		return nil
	}

	containers, found, err := unstructured.NestedSlice(unstructuredObj.Object, "spec", "template", "spec", "containers")
	if err != nil {
		klog.Errorf("failed to get nested slice : %s", err.Error())
		return err
	}

	if found {
		for _, container := range containers {
			if disableLivenessProbe{
				unstructured.RemoveNestedField(container.(map[string]interface{}),  "livenessProbe")
			}
			if disableReadinessProbe{
				unstructured.RemoveNestedField(container.(map[string]interface{}),  "readinessProbe")
			}
		}
		err = unstructured.SetNestedSlice(unstructuredObj.Object, containers, "spec", "template", "spec", "containers")
		if err != nil {
			klog.Errorf("failed to set nested slice : %s", err.Error())
			return err
		}
	}
	return nil
}

func buildProbeStatus(unstructuredObj *unstructured.Unstructured, customHealthProbeArgs *CustomHealthProbeArgs) (disableLivenessProbe bool,disableReadinessProbe bool) {
	if unstructuredObj == nil || customHealthProbeArgs == nil {
		return
	}

	if customHealthProbeArgs.DisableAllLivenessProbe {
		disableLivenessProbe = true
	} else {
		for _, probeResource := range customHealthProbeArgs.DisableLivenessProbeResourceList {
			if probeResource.Kind == unstructuredObj.GetKind() && probeResource.ResourceName == unstructuredObj.GetName() {
				disableLivenessProbe = true
				break
			}
		}
	}

	if customHealthProbeArgs.DisableAllReadinessProbe {
		disableReadinessProbe = true
	} else {
		for _, probeResource := range customHealthProbeArgs.DisableReadinessProbeResourceList {
			if probeResource.Kind == unstructuredObj.GetKind() && probeResource.ResourceName == unstructuredObj.GetName() {
				disableReadinessProbe = true
				break
			}
		}
	}

	return
}