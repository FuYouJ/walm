package plugins

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"github.com/tidwall/sjson"
	"k8s.io/api/apps/v1"
	"WarpCloud/walm/pkg/util"
	"transwarp/isomateset-client/pkg/apis/apiextensions.transwarp.io/v1alpha1"
)

func convertToUnstructured(obj runtime.Object) (runtime.Object, error) {
	var unstructuredObj unstructured.Unstructured
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return unstructuredObj.DeepCopyObject(), err
	}
	objMap := make(map[string]interface{}, 0)
	err = json.Unmarshal(objBytes, &objMap)
	if err != nil {
		return unstructuredObj.DeepCopyObject(), err
	}

	unstructuredObj.SetUnstructuredContent(objMap)
	return unstructuredObj.DeepCopyObject(), nil
}

func addNestedStringMap(obj map[string]interface{}, stringMapToAdd map[string]string, fields ...string) error {
	if len(stringMapToAdd) == 0 {
		return nil
	}
	stringMap, _, err := unstructured.NestedStringMap(obj, fields...)
	if err != nil {
		klog.Errorf("failed to get string map : %s", err.Error())
		return err
	}
	if stringMap == nil {
		stringMap = map[string]string{}
	}
	for k, v := range stringMapToAdd {
		stringMap[k] = v
	}
	err = unstructured.SetNestedStringMap(obj, stringMap, fields...)
	if err != nil {
		klog.Errorf("failed to set string map : %s", err.Error())
		return err
	}
	return nil
}

func addNestedSliceObj(obj map[string]interface{}, sliceObjToAdd []interface{}, fields ...string) error {
	if len(sliceObjToAdd) == 0 {
		return nil
	}

	sliceToAdd := []interface{}{}
	for _, obj := range sliceObjToAdd {
		objMap, err := util.ConvertObjectToJsonMap(obj)
		if err != nil {
			klog.Errorf("failed to convert obj to json map : %s", err.Error())
			return err
		}
		sliceToAdd = append(sliceToAdd, objMap)
	}

	slice, _, err := unstructured.NestedSlice(obj, fields...)
	if err != nil {
		klog.Errorf("failed to get slice : %s", err.Error())
		return err
	}

	slice = append(slice, sliceToAdd...)
	err = unstructured.SetNestedSlice(obj, slice, fields...)
	if err != nil {
		klog.Errorf("failed to set slice : %s", err.Error())
		return err
	}
	return nil
}

func convertUnstructuredToSts(unstructured *unstructured.Unstructured) (*v1.StatefulSet, error) {
	resourceBytes, err := json.Marshal(unstructured.Object)
	if err != nil {
		klog.Errorf("failed to marshal : %s", err.Error())
		return nil, err
	}

	resourceJson := string(resourceBytes)
	resourceJson, err = sjson.Set(resourceJson, "apiVersion", "apps/v1")
	if err != nil {
		klog.Errorf("failed to set apiversion to apps/v1 : %s", err.Error())
		return nil, err
	}

	sts := &v1.StatefulSet{}
	err = json.Unmarshal([]byte(resourceJson), sts)
	if err != nil {
		klog.Errorf("failed to unmarshal : %s", err.Error())
		return nil, err
	}
	return sts, nil
}

func convertUnstructuredToIsomateSet(unstructured *unstructured.Unstructured) (*v1alpha1.IsomateSet, error) {
	resourceBytes, err := json.Marshal(unstructured.Object)
	if err != nil {
		klog.Errorf("failed to marshal : %s", err.Error())
		return nil, err
	}

	isomateSet := &v1alpha1.IsomateSet{}
	err = json.Unmarshal(resourceBytes, isomateSet)
	if err != nil {
		klog.Errorf("failed to unmarshal : %s", err.Error())
		return nil, err
	}
	return isomateSet, nil
}

func MergeIsomateSets(iso1, iso2 runtime.Object) (runtime.Object, error) {
	convertedIso1, err := convertUnstructuredToIsomateSet(iso1.(*unstructured.Unstructured))
	if err != nil {
		return nil, err
	}
	convertedIso2, err := convertUnstructuredToIsomateSet(iso2.(*unstructured.Unstructured))
	if err != nil {
		return nil, err
	}

	for key, value := range convertedIso2.Spec.VersionTemplates {
		convertedIso1.Spec.VersionTemplates[key] = value
	}

	return convertToUnstructured(convertedIso1)
}