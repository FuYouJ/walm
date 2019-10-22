package plugins

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"encoding/json"
)

const (
	PauseReleasePluginName = "PauseRelease"
	UsePodOfflineKey       = "Transwarp_Walm_Use_Pod_Offline"
	UsePodOfflineValue     = "true"
	PodOfflineKey          = "offline-pod.transwarp.io/all-ordinals"
)

func init() {
	register(PauseReleasePluginName, &WalmPluginRunner{
		Run:  PauseRelease,
		Type: Pre_Install,
	})
}

func PauseRelease(context *PluginContext, args string) (err error) {
	for _, resource := range context.Resources {
		unstructuredObj := resource.(*unstructured.Unstructured)
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Deployment":
			err := scaleReplicasToZero(unstructuredObj.Object)
			if err != nil {
				klog.Errorf("failed to scale replicas to 0 : %s", err.Error())
				return err
			}
		case "StatefulSet":
			annos := unstructuredObj.GetAnnotations()
			if len(annos) > 0 && annos[UsePodOfflineKey] == UsePodOfflineValue {
				annos[PodOfflineKey] = ""
				unstructuredObj.SetAnnotations(annos)
			} else {
				err :=scaleReplicasToZero(unstructuredObj.Object)
				if err != nil {
					klog.Errorf("failed to scale replicas to 0 : %s", err.Error())
					return err
				}
			}
		}
	}
	return
}

func scaleReplicasToZero(obj map[string]interface{}) error{
	err := unstructured.SetNestedField(obj, json.Number("0"), "spec", "replicas")
	if err != nil {
		klog.Errorf("failed to set nested field : %s", err.Error())
		return err
	}
	return nil
}