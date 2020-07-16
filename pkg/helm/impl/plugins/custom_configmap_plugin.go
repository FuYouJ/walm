package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"strings"
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
	VolumeMountPath   string		   `json:"volumeMountPath"`
	ContainerName     string           `json:"containerName"`
	Items             []*AddConfigItem `json:"items"`
}

type AddConfigItem struct {
	ConfigMapData                  string `json:"configMapData"`
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

	for _, resource := range context.Resources {
		unstructuredObj := resource.(*unstructured.Unstructured)
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Job", "Deployment", "DaemonSet", "StatefulSet":
			for configMapName, addConfigMapObj := range customConfigmapArgs.ConfigmapToAdd {
				err = mountConfigMap(unstructuredObj, context.R.Name, configMapName, addConfigMapObj)
				if err != nil {
					klog.Errorf("mountConfigMap %s %s %v error %v", context.R.Name, configMapName, *addConfigMapObj, err)
					return err
				}
			}
		case "Configmap":
			if isSkippedConfigMap(unstructuredObj.GetName(), customConfigmapArgs) {
				err = addNestedStringMap(unstructuredObj.Object, map[string]string{ResourceUpgradePolicyAnno: UpgradePolicy}, "metadata", "annotations")
				if err != nil {
					klog.Errorf("failed add nested string map : %s", err.Error())
					return err
				}
			}
		}
	}

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
		context.Resources = append(context.Resources, unstructuredObj)
	}

	return
}

func isSkippedConfigMap(name string, args *CustomConfigmapArgs) bool{
	if args.ConfigmapSkipAll == true {
		return true
	} else {
		for _, skipConfigmapName := range args.ConfigmapToSkipNames {
			if skipConfigmapName == name {
				return true
			}
		}
		return false
	}
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
		if item.ConfigMapVolumeMountsSubPath != "" {
			token := strings.Split(item.ConfigMapVolumeMountsSubPath, "/")
			configMapObj.Data[token[len(token) - 1]] = item.ConfigMapData
		}
	}

	return configMapObj, nil
}

func splitConfigmapVolumes(releaseName, configMapName string, addConfigMapObj *AddConfigmapObject) (v1.Volume, v1.VolumeMount, error) {
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

	for _, addConfigItem := range addConfigMapObj.Items {
		token := strings.Split(addConfigItem.ConfigMapVolumeMountsSubPath, "/")
		configMapVolume.VolumeSource.ConfigMap.Items = append(configMapVolume.VolumeSource.ConfigMap.Items, v1.KeyToPath{
			Key:  token[len(token) - 1],
			Path: addConfigItem.ConfigMapVolumeMountsSubPath,
		})
	}

	configMapVolumeMounts := v1.VolumeMount{
		Name:      fmt.Sprintf("walmplugin-%s-%s-cm", configMapName, releaseName),
		MountPath: addConfigMapObj.VolumeMountPath,
	}

	return configMapVolume, configMapVolumeMounts, nil
}

func mountConfigMap(unstructuredObj *unstructured.Unstructured, releaseName, configMapName string, addConfigMapObj *AddConfigmapObject) error {
	resourceKind := unstructuredObj.GetKind()
	resourceName := unstructuredObj.GetName()
	if !addConfigMapObj.ApplyAllResources {
		if addConfigMapObj.Kind != resourceKind || addConfigMapObj.ResourceName != resourceName {
			return nil
		}
	}

	configMapVolume, configMapVolumeMounts, err := splitConfigmapVolumes(releaseName, configMapName, addConfigMapObj)
	if err != nil {
		klog.Errorf("failed to split config map volumes : %s", err.Error())
		return err
	}

	err = addNestedSliceObj(unstructuredObj.Object, []interface{}{
		configMapVolume,
	}, "spec", "template", "spec", "volumes")
	if err != nil {
		klog.Errorf("failed to add nested slice objs : %s", err.Error())
		return err
	}

	containers, found, err := unstructured.NestedSlice(unstructuredObj.Object, "spec", "template", "spec", "containers")
	if err != nil {
		klog.Errorf("failed to get containers %s", err.Error())
		return err
	}

	var k8sContainers []v1.Container
	containersData, err := json.Marshal(containers)
	if err != nil {
		klog.Errorf("failed to marshal containers type interface to []byte : %s", err.Error())
		return err
	}
	err = json.Unmarshal(containersData, &k8sContainers)
	if err != nil {
		klog.Errorf("failed to unmarshal containers type []byte to []corev1.Container : %s", err.Error())
		return err
	}

	existMountPaths := getExistMountPaths(k8sContainers)
	if existMountPaths[configMapVolumeMounts.MountPath] != "" {
			return errors.Errorf("volumeMountPath %s already exist in containers, duplicated with volume mount name %s", configMapVolumeMounts.Name, existMountPaths[configMapVolumeMounts.MountPath])
	}

	if found {
		for _, container := range containers {
			configMapVolumeMountsInterface := []interface{}{}
			configMapVolumeMountsInterface = append(configMapVolumeMountsInterface, configMapVolumeMounts)
			err = addNestedSliceObj(container.(map[string]interface{}), configMapVolumeMountsInterface, "volumeMounts")
			if err != nil {
				klog.Errorf("failed to add nested slice obj : %s", err.Error())
				return err
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

func getExistMountPaths(containers []v1.Container) map[string]string {
	existMountPaths := map[string]string{}
	for _, container := range containers {
		for _, volumeMount := range container.VolumeMounts {
			existMountPaths[volumeMount.MountPath] = volumeMount.Name
		}
	}
	return existMountPaths
}
