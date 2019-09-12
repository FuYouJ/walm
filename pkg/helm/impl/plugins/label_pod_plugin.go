package plugins

import (
	"encoding/json"
	"github.com/tidwall/sjson"
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
			newResource = append(newResource, job)
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
			newResource = append(newResource, deployment)
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
			newResource = append(newResource, daemonSet)
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
			newResource = append(newResource, statefulSet)
		default:
			newResource = append(newResource, resource)
		}
	}
	context.Resources = newResource
	return
}

func buildStatefulSet(obj runtime.Object) (*appsv1.StatefulSet, error) {
	if statefulSet, ok := obj.(*appsv1.StatefulSet); ok {
		return statefulSet, nil
	} else {
		objBytes, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		objStr := string(objBytes)
		objStr, err = sjson.Set(objStr, "apiVersion", "apps/v1")
		if err != nil {
			return nil, err
		}
		statefulSet = &appsv1.StatefulSet{}
		err = json.Unmarshal([]byte(objStr), statefulSet)
		if err != nil {
			return nil, err
		}
		return statefulSet, nil
	}
}

func buildJob(obj runtime.Object) (*batchv1.Job, error) {
	if job, ok := obj.(*batchv1.Job); ok {
		return job, nil
	} else {
		objBytes, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		objStr := string(objBytes)
		objStr, err = sjson.Set(objStr, "apiVersion", "batch/v1")
		if err != nil {
			return nil, err
		}
		job = &batchv1.Job{}
		err = json.Unmarshal([]byte(objStr), job)
		if err != nil {
			return nil, err
		}
		return job, nil
	}
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

func buildDaemonSet(obj runtime.Object) (*appsv1.DaemonSet, error) {
	if daemonSet, ok := obj.(*appsv1.DaemonSet); ok {
		return daemonSet, nil
	} else {
		objBytes, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		objStr := string(objBytes)
		objStr, err = sjson.Set(objStr, "apiVersion", "apps/v1")
		if err != nil {
			return nil, err
		}
		daemonSet = &appsv1.DaemonSet{}
		err = json.Unmarshal([]byte(objStr), daemonSet)
		if err != nil {
			return nil, err
		}
		return daemonSet, nil
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

func buildDeployment(obj runtime.Object) (*appsv1.Deployment, error) {
	if deployment, ok := obj.(*appsv1.Deployment); ok {
		return deployment, nil
	} else {
		objBytes, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		objStr := string(objBytes)
		objStr, err = sjson.Set(objStr, "apiVersion", "apps/v1")
		if err != nil {
			return nil, err
		}
		deployment = &appsv1.Deployment{}
		err = json.Unmarshal([]byte(objStr), deployment)
		if err != nil {
			return nil, err
		}
		return deployment, nil
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
