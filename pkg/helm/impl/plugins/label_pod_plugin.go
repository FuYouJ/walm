package plugins

import (
	"encoding/json"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

	newResource := []runtime.Object{}
	for _, resource := range context.Resources {
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Job":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Infof("failed to convert unstructured : %s", err.Error())
				return err
			}
			job, err := buildJob(converted)
			if err != nil {
				klog.Infof("failed to build deployment : %s", err.Error())
				return err
			}
			labelJobPod(job, labelPodArgs)
			unstructuredObj, err := convertToUnstructured(job)
			if err != nil {
				klog.Infof("failed to convertToUnstructured : %v", *job)
				return err
			}
			newResource = append(newResource, unstructuredObj)
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
			labelDeploymentPod(deployment, labelPodArgs)
			unstructuredObj, err := convertToUnstructured(deployment)
			if err != nil {
				klog.Infof("failed to convertToUnstructured : %v", *deployment)
				return err
			}
			newResource = append(newResource, unstructuredObj)
		case "DaemonSet":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Infof("failed to convert unstructured : %s", err.Error())
				return err
			}
			daemonSet, err := buildDaemonSet(converted)
			if err != nil {
				klog.Infof("failed to build daemonSet : %s", err.Error())
				return err
			}
			labelDaemonSetPod(daemonSet, labelPodArgs)
			unstructuredObj, err := convertToUnstructured(daemonSet)
			if err != nil {
				klog.Infof("failed to convertToUnstructured : %v", *daemonSet)
				return err
			}
			newResource = append(newResource, unstructuredObj)
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
			labelStatefulSetPod(statefulSet, labelPodArgs)
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
	return
}

func labelStatefulSetPod(statefulSet *appsv1.StatefulSet, labelPodArgs *LabelPodArgs) {
	if statefulSet.Spec.Template.Labels == nil {
		statefulSet.Spec.Template.Labels = labelPodArgs.LabelsToAdd
	} else {
		for k, v := range labelPodArgs.LabelsToAdd {
			statefulSet.Spec.Template.Labels[k] = v
		}
	}
	if statefulSet.Spec.Template.Annotations == nil {
		statefulSet.Spec.Template.Annotations = labelPodArgs.AnnotationsToAdd
	} else {
		for k, v := range labelPodArgs.AnnotationsToAdd {
			statefulSet.Spec.Template.Annotations[k] = v
		}
	}
}

func labelJobPod(job *batchv1.Job, labelPodArgs *LabelPodArgs) {
	if job.Spec.Template.Labels == nil {
		job.Spec.Template.Labels = labelPodArgs.LabelsToAdd
	} else {
		for k, v := range labelPodArgs.LabelsToAdd {
			job.Spec.Template.Labels[k] = v
		}
	}
	if job.Spec.Template.Annotations == nil {
		job.Spec.Template.Annotations = labelPodArgs.AnnotationsToAdd
	} else {
		for k, v := range labelPodArgs.AnnotationsToAdd {
			job.Spec.Template.Annotations[k] = v
		}
	}
}

func labelDaemonSetPod(daemonSet *appsv1.DaemonSet, labelPodArgs *LabelPodArgs) {
	if daemonSet.Spec.Template.Labels == nil {
		daemonSet.Spec.Template.Labels = labelPodArgs.LabelsToAdd
	} else {
		for k, v := range labelPodArgs.LabelsToAdd {
			daemonSet.Spec.Template.Labels[k] = v
		}
	}
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = labelPodArgs.AnnotationsToAdd
	} else {
		for k, v := range labelPodArgs.AnnotationsToAdd {
			daemonSet.Spec.Template.Annotations[k] = v
		}
	}
}

func labelDeploymentPod(deployment *appsv1.Deployment, labelPodArgs *LabelPodArgs) {
	if deployment.Spec.Template.Labels == nil {
		deployment.Spec.Template.Labels = labelPodArgs.LabelsToAdd
	} else {
		for k, v := range labelPodArgs.LabelsToAdd {
			deployment.Spec.Template.Labels[k] = v
		}
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = labelPodArgs.AnnotationsToAdd
	} else {
		for k, v := range labelPodArgs.AnnotationsToAdd {
			deployment.Spec.Template.Annotations[k] = v
		}
	}
}
