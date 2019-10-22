package plugins

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
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
		objMap, err := convertObjToJsonMap(obj)
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

func convertObjToJsonMap(obj interface{}) (map[string]interface{}, error) {
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	objMap := map[string]interface{}{}
	err = json.Unmarshal(objBytes, &objMap)
	if err != nil {
		return nil, err
	}
	return objMap, nil
}