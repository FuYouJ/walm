package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
)

const (
	CustomConfigmapPluginName = "CustomConfigmap"
)

func init() {
	register(CustomConfigmapPluginName, &WalmPluginRunner{
		Run:  CustomConfigmapTransform,
		Type: Pre_Install,
	})
}

type AddConfigmapObject struct {
	ApplyAllResources bool             `json:"applyAllResources"`
	Kind              string           `json:"kind"`
	ResourceName      string           `json:"resourceName"`
	ContainerName     string           `json:"containerName"`
	Items             []*AddConfigItem `json:"items"`
}

type AddConfigItem struct {
	ConfigMapData                  string `json:"configMapData"`
	ConfigMapVolumeMountsMountPath string `json:"configMapVolumeMountsMountPath"`
	ConfigMapVolumeMountsSubPath   string `json:"configMapVolumeMountsSubPath"`
	ConfigMapMode                  int32  `json:"configMapMode"`
}

type CustomConfigmapArgs struct {
	ConfigmapToAdd       map[string]*AddConfigmapObject `json:"configmapToAdd" description:"add extra configmap"`
	ConfigmapToSkipNames []string                       `json:"configmapToSkipNames" description:"upgrade skip to render configmap"`
	ConfigmapSkipAll     bool                           `json:"configmapSkipAll" description:"upgrade skip all configmap resources"`
}

func CustomConfigmapTransform(context *PluginContext, args string) (err error) {
	if args == "" {
		klog.Infof("ignore labeling pod, because plugin args is empty")
		return nil
	} else {
		klog.Infof("label pod args : %s", args)
	}
	customConfigmapArgs := &CustomConfigmapArgs{}
	err = json.Unmarshal([]byte(args), customConfigmapArgs)
	if err != nil {
		klog.Infof("failed to unmarshal plugin args : %s", err.Error())
		return err
	}

	newResource := []runtime.Object{}
	chartConfigmapResources := make([]*v1.ConfigMap, 0)
	// ToDo: Add Configmap volume/volumeMounts to Resources
	for configMapName, addObj := range customConfigmapArgs.ConfigmapToAdd {
		configMapObj, err := convertK8SConfigMap(context.R.Name, context.R.Namespace, configMapName, addObj)
		if err != nil {
			klog.Errorf("add configMap plugin error %v", err)
			continue
		}
		unstructuredObj, err := convertToUnstructured(configMapObj)
		if err != nil {
			klog.Infof("failed to convertToUnstructured : %v", *configMapObj)
			return err
		}
		newResource = append(newResource, unstructuredObj)
	}

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
				klog.Infof("failed to build Job : %s", err.Error())
				return err
			}
			for configMapName, addConfigMapObj := range customConfigmapArgs.ConfigmapToAdd {
				err = addConfigMapJob(context.R.Name, configMapName, job, addConfigMapObj)
				if err != nil {
					klog.Errorf("addConfigMapJob %s %s %v error %v", context.R.Name, configMapName, *addConfigMapObj, err)
					return err
				}
			}
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
			for configMapName, addConfigMapObj := range customConfigmapArgs.ConfigmapToAdd {
				err = addConfigMapDeployment(context.R.Name, configMapName, deployment, addConfigMapObj)
				if err != nil {
					klog.Errorf("addConfigMapDeployment %s %s %v error %v", context.R.Name, configMapName, *addConfigMapObj, err)
					return err
				}
			}
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
			for configMapName, addConfigMapObj := range customConfigmapArgs.ConfigmapToAdd {
				err = addConfigMapDaemonSet(context.R.Name, configMapName, daemonSet, addConfigMapObj)
				if err != nil {
					klog.Errorf("addConfigMapDaemonSet %s %s %v error %v", context.R.Name, configMapName, *addConfigMapObj, err)
					return err
				}
			}
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
			for configMapName, addConfigMapObj := range customConfigmapArgs.ConfigmapToAdd {
				err = addConfigMapStatefulSet(context.R.Name, configMapName, statefulSet, addConfigMapObj)
				if err != nil {
					klog.Errorf("addConfigMapStatefulSet %s %s %v error %v", context.R.Name, configMapName, *addConfigMapObj, err)
					return err
				}
			}
			unstructuredObj, err := convertToUnstructured(statefulSet)
			if err != nil {
				klog.Infof("failed to convertToUnstructured : %v", *statefulSet)
				return err
			}
			newResource = append(newResource, unstructuredObj)
		case "Configmap":
			converted, err := convertUnstructured(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Infof("failed to convert unstructured : %s", err.Error())
				return err
			}
			convertedConfigMap, err := buildConfigmap(converted)
			if err != nil {
				klog.Errorf("buildConfigmap %v error %v", converted, err)
				return err
			}
			if convertedConfigMap.Annotations != nil {
				customAnnotation, ok := convertedConfigMap.Annotations["transwarp/walmplugin.custom.configmap"]
				if ok && customAnnotation == "true" {
					continue
				}
			}
			chartConfigmapResources = append(chartConfigmapResources, convertedConfigMap)
		default:
			newResource = append(newResource, resource)
		}
	}

	klog.Infof("%s enabled plugin %s", context.R.Name, CustomConfigmapPluginName)
	for _, chartConfigmapResource := range chartConfigmapResources {
		if chartConfigmapResource.Annotations == nil {
			chartConfigmapResource.Annotations = make(map[string]string, 0)
		}
		if customConfigmapArgs.ConfigmapSkipAll == true {
			chartConfigmapResource.Annotations[ResourceUpgradePolicyAnno] = UpgradePolicy
		} else {
			for _, skipConfigmapName := range customConfigmapArgs.ConfigmapToSkipNames {
				if skipConfigmapName == chartConfigmapResource.Name {
					chartConfigmapResource.Annotations[ResourceUpgradePolicyAnno] = UpgradePolicy
					break
				}
			}
		}
		unstructuredObj, err := convertToUnstructured(chartConfigmapResource)
		if err != nil {
			klog.Infof("failed to convertToUnstructured : %v", *chartConfigmapResource)
			return err
		}
		newResource = append(newResource, unstructuredObj)
	}

	context.Resources = newResource
	return
}

func convertK8SConfigMap(releaseName, releaseNamespace, configMapName string, addObj *AddConfigmapObject) (*v1.ConfigMap, error) {
	if len(configMapName) == 0 || len(configMapName) > qualifiedNameMaxLength || !qualifiedNameRegexp.MatchString(configMapName) {
		return nil, errors.New(fmt.Sprintf("invaild configmap name %s", configMapName))
	}

	configMapObj := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
	}
	configMapObj.SetName(fmt.Sprintf("walmplugin-%s-%s-cm", configMapName, releaseName))
	configMapObj.SetNamespace(releaseNamespace)
	configMapObj.SetAnnotations(map[string]string{
		"transwarp/walmplugin.custom.configmap": "true",
	})
	configMapObj.SetLabels(map[string]string{
		"release":  releaseName,
		"heritage": "walmplugin",
	})
	configMapObj.Data = make(map[string]string, 0)
	for _, item := range addObj.Items {
		configMapObj.Data[item.ConfigMapVolumeMountsSubPath] = item.ConfigMapData
	}

	return configMapObj, nil
}

func splitConfigmapVolumes(releaseName, configMapName string, addConfigMapObj *AddConfigmapObject) (v1.Volume, []v1.VolumeMount, error) {
	// ToDo: Add Params Validate

	//ConfigMapVolumeSource
	configMapVolume := v1.Volume{
		Name: fmt.Sprintf("walmplugin-%s-%s-cm", configMapName, releaseName),
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: fmt.Sprintf("walmplugin-%s-%s-cm", configMapName, releaseName),
				},
			},
		},
	}
	configMapVolumeMounts := make([]v1.VolumeMount, 0)
	for _, addConfigItem := range addConfigMapObj.Items {
		configMapVolume.VolumeSource.ConfigMap.Items = append(configMapVolume.VolumeSource.ConfigMap.Items, v1.KeyToPath{
			Key:  addConfigItem.ConfigMapVolumeMountsSubPath,
			Path: addConfigItem.ConfigMapVolumeMountsSubPath,
		})

		configMapVolumeMounts = append(configMapVolumeMounts, v1.VolumeMount{
			Name:      fmt.Sprintf("walmplugin-%s-%s-cm", configMapName, releaseName),
			MountPath: addConfigItem.ConfigMapVolumeMountsMountPath,
			SubPath:   addConfigItem.ConfigMapVolumeMountsSubPath,
		})
	}

	return configMapVolume, configMapVolumeMounts, nil
}

func addConfigMapStatefulSet(releaseName, configMapName string, statefulSet *appsv1.StatefulSet, addConfigMapObj *AddConfigmapObject) error {
	if !addConfigMapObj.ApplyAllResources {
		if addConfigMapObj.Kind != "StatefulSet" || addConfigMapObj.ResourceName != statefulSet.Name {
			return nil
		}
	}
	configMapVolume, configMapVolumeMounts, err := splitConfigmapVolumes(releaseName, configMapName, addConfigMapObj)
	if err != nil {
		return err
	}
	if statefulSet.Spec.Template.Spec.Volumes == nil {
		statefulSet.Spec.Template.Spec.Volumes = []v1.Volume{
			configMapVolume,
		}
	} else {
		statefulSet.Spec.Template.Spec.Volumes = append(statefulSet.Spec.Template.Spec.Volumes, configMapVolume)
	}
	for idx, _ := range statefulSet.Spec.Template.Spec.Containers {
		if statefulSet.Spec.Template.Spec.Containers[idx].VolumeMounts == nil {
			statefulSet.Spec.Template.Spec.Containers[idx].VolumeMounts = configMapVolumeMounts
		} else {
			statefulSet.Spec.Template.Spec.Containers[idx].VolumeMounts = append(statefulSet.Spec.Template.Spec.Containers[idx].VolumeMounts, configMapVolumeMounts...)
		}
	}
	return nil
}

func addConfigMapJob(releaseName, configMapName string, job *batchv1.Job, addConfigMapObj *AddConfigmapObject) error {
	if !addConfigMapObj.ApplyAllResources {
		if addConfigMapObj.Kind != "Job" || addConfigMapObj.ResourceName != job.Name {
			return nil
		}
	}
	configMapVolume, configMapVolumeMounts, err := splitConfigmapVolumes(releaseName, configMapName, addConfigMapObj)
	if err != nil {
		return err
	}
	if job.Spec.Template.Spec.Volumes == nil {
		job.Spec.Template.Spec.Volumes = []v1.Volume{
			configMapVolume,
		}
	} else {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, configMapVolume)
	}
	for idx, _ := range job.Spec.Template.Spec.Containers {
		if job.Spec.Template.Spec.Containers[idx].VolumeMounts == nil {
			job.Spec.Template.Spec.Containers[idx].VolumeMounts = configMapVolumeMounts
		} else {
			job.Spec.Template.Spec.Containers[idx].VolumeMounts = append(job.Spec.Template.Spec.Containers[idx].VolumeMounts, configMapVolumeMounts...)
		}
	}
	return nil
}

func addConfigMapDeployment(releaseName, configMapName string, deployment *appsv1.Deployment, addConfigMapObj *AddConfigmapObject) error {
	if !addConfigMapObj.ApplyAllResources {
		if addConfigMapObj.Kind != "Deployment" || addConfigMapObj.ResourceName != deployment.Name {
			return nil
		}
	}
	configMapVolume, configMapVolumeMounts, err := splitConfigmapVolumes(releaseName, configMapName, addConfigMapObj)
	if err != nil {
		return err
	}
	if deployment.Spec.Template.Spec.Volumes == nil {
		deployment.Spec.Template.Spec.Volumes = []v1.Volume{
			configMapVolume,
		}
	} else {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, configMapVolume)
	}
	for idx, _ := range deployment.Spec.Template.Spec.Containers {
		if deployment.Spec.Template.Spec.Containers[idx].VolumeMounts == nil {
			deployment.Spec.Template.Spec.Containers[idx].VolumeMounts = configMapVolumeMounts
		} else {
			deployment.Spec.Template.Spec.Containers[idx].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[idx].VolumeMounts, configMapVolumeMounts...)
		}
	}
	return nil
}

func addConfigMapDaemonSet(releaseName, configMapName string, daemonSet *appsv1.DaemonSet, addConfigMapObj *AddConfigmapObject) error {
	if !addConfigMapObj.ApplyAllResources {
		if addConfigMapObj.Kind != "DaemonSet" || addConfigMapObj.ResourceName != daemonSet.Name {
			return nil
		}
	}
	configMapVolume, configMapVolumeMounts, err := splitConfigmapVolumes(releaseName, configMapName, addConfigMapObj)
	if err != nil {
		return err
	}
	if daemonSet.Spec.Template.Spec.Volumes == nil {
		daemonSet.Spec.Template.Spec.Volumes = []v1.Volume{
			configMapVolume,
		}
	} else {
		daemonSet.Spec.Template.Spec.Volumes = append(daemonSet.Spec.Template.Spec.Volumes, configMapVolume)
	}
	for idx, _ := range daemonSet.Spec.Template.Spec.Containers {
		if daemonSet.Spec.Template.Spec.Containers[idx].VolumeMounts == nil {
			daemonSet.Spec.Template.Spec.Containers[idx].VolumeMounts = configMapVolumeMounts
		} else {
			daemonSet.Spec.Template.Spec.Containers[idx].VolumeMounts = append(daemonSet.Spec.Template.Spec.Containers[idx].VolumeMounts, configMapVolumeMounts...)
		}
	}
	return nil
}
