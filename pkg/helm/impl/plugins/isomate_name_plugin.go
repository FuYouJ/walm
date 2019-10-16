package plugins

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"strings"
)

const (
	NeedIsomateNameAnnoationKey = "NeedIsomateName"
	NeedIsomateNameAnnoationValue = "true"

	IsomateNameLabelKey = "IsomateName"
	IsomateNamePluginName = "IsomateName"
)

// IsomateName plugin is used to modify isomate resources name
func init() {
	register(IsomateNamePluginName, &WalmPluginRunner{
		Run:  IsomateName,
		Type: Pre_Install,
	})
}

type IsomateNameArgs struct {
	Name string `json:"name" description:"isomate name"`
}

func IsomateName(context *PluginContext, args string) error {
	if args == "" {
		klog.Infof("ignore isomate name plugin, because plugin args is empty")
		return nil
	} else {
		klog.Infof("isomate name plugin args : %s", args)
	}
	isomateNameArgs := &IsomateNameArgs{}
	err := json.Unmarshal([]byte(args), isomateNameArgs)
	if err != nil {
		klog.Infof("failed to unmarshal plugin args : %s", err.Error())
		return err
	}

	if isomateNameArgs.Name != "" {
		for _, resource := range context.Resources {
			unstructured := resource.(*unstructured.Unstructured)
			annos := unstructured.GetAnnotations()
			if len(annos) > 0 && strings.ToLower(annos[NeedIsomateNameAnnoationKey]) == NeedIsomateNameAnnoationValue {
				unstructured.SetName(buildResourceName(unstructured, isomateNameArgs.Name))
				labels := unstructured.GetLabels()
				if labels == nil {
					labels = map[string]string{}
				}
				labels[IsomateNameLabelKey] = isomateNameArgs.Name
				unstructured.SetLabels(labels)
			}
		}
	}

	return nil
}

func buildResourceName(unstructured *unstructured.Unstructured, isomateName string) string {
	return unstructured.GetName() + "-" + isomateName
}

