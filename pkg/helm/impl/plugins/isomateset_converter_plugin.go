package plugins

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	"k8s.io/klog"
	"transwarp/isomateset-client/pkg/convert"
	"WarpCloud/walm/pkg/util"
)

const (
	IsomateSetConverterPluginName = "IsomateSetConverter"
	ConvertToIsoamteSetLabelKey   = "ConvertToIsomateSet"
	ConvertToIsoamteSetLabelValue = "true"
)

// IsomateName plugin is used to modify isomate resources name
func init() {
	register(IsomateSetConverterPluginName, &WalmPluginRunner{
		Run:  ConvertStsToIsomateSet,
		Type: Pre_Install,
	})
}

func ConvertStsToIsomateSet(context *PluginContext, args string) error {
	for _, resource := range context.Resources {
		unstructured := resource.(*unstructured.Unstructured)
		annos := unstructured.GetAnnotations()
		if len(annos) > 0 && strings.ToLower(annos[ConvertToIsoamteSetLabelKey]) == ConvertToIsoamteSetLabelValue {

			sts, err := convertUnstructuredToSts(unstructured)
			if err != nil {
				klog.Errorf("failed to convert unstructured to stateful set : %s", err.Error())
				return err
			}

			isomateSet, err := v1alpha1.Convert_StatefulSets_To_v1alpha1_IsomateSet(sts)
			if err != nil {
				klog.Errorf("failed to convert sts to isomate set : %s", err.Error())
				return err
			}

			isomateSetJsonMap, err := util.ConvertObjectToJsonMap(isomateSet)
			if err != nil {
				klog.Errorf("failed to convert isomate set to json map : %s", err.Error())
				return err
			}
			unstructured.Object = isomateSetJsonMap
			klog.Infof("succeed converting sts %s/%s to isomate set", unstructured.GetNamespace(), unstructured.GetName())
		}
	}

	return nil
}

