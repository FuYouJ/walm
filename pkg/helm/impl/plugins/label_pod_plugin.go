package plugins

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
)

const (
	LabelPodPluginName = "LabelPod"
)

func init() {
	register(LabelPodPluginName, &WalmPluginRunner{
		Run:  LabelPod,
		Type: Pre_Install,
	})
}

type LabelPodArgs struct {
	LabelsToAdd      map[string]string `json:"labelsToAdd" description:"labels to add"`
	AnnotationsToAdd map[string]string `json:"annotationsToAdd" description:"annotations to add"`
}

func LabelPod(context *PluginContext, args string) (err error) {
	if args == "" {
		klog.Infof("ignore labeling pod, because plugin args is empty")
		return nil
	} else {
		klog.Infof("label pod args : %s", args)
	}
	labelPodArgs := &LabelPodArgs{}
	err = json.Unmarshal([]byte(args), labelPodArgs)
	if err != nil {
		klog.Infof("failed to unmarshal plugin args : %s", err.Error())
		return err
	}

	if len(labelPodArgs.LabelsToAdd) == 0 && len(labelPodArgs.AnnotationsToAdd) == 0 {
		klog.Warningf("nothing to do")
		return nil
	}

	for _, resource := range context.Resources {
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Job", "Deployment", "DaemonSet", "StatefulSet":
			unstruct := resource.(*unstructured.Unstructured)
			err := setNestedStringMap(unstruct.Object, labelPodArgs.LabelsToAdd, "spec", "template", "metadata", "labels")
			if err != nil {
				klog.Errorf("failed to add labels to pod : %s", err.Error())
				return err
			}
			err = setNestedStringMap(unstruct.Object, labelPodArgs.AnnotationsToAdd, "spec", "template", "metadata", "annotations")
			if err != nil {
				klog.Errorf("failed to add labels to pod : %s", err.Error())
				return err
			}
		}
	}
	return
}
