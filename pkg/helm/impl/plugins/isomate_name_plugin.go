package plugins

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"strings"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/util"
)

const (
	NeedIsomateNameAnnoationKey   = "NeedIsomateName"
	NeedIsomateNameAnnoationValue = "true"

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
	Name           string `json:"name" description:"isomate name"`
	DefaultIsomate bool   `json:"defaultIsomate" description:"default isomate"`
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
			isomateResourceName := buildResourceName(unstructured, isomateNameArgs.Name)
			if unstructured.GetKind() == string(k8s.IsomateSetKind) {
				isomateSet, err := convertUnstructuredToIsomateSet(unstructured)
				if err != nil {
					klog.Errorf("failed to convert unstructured to isomate set : %s", err.Error())
					return err
				}
				isoName := isomateSet.Name
				versionTemplate := isomateSet.Spec.VersionTemplates[isoName]
				if versionTemplate != nil {
					if versionTemplate.Labels == nil {
						versionTemplate.Labels = map[string]string{}
					}
					versionTemplate.Labels[k8s.IsomateNameLabelKey] = isomateNameArgs.Name
					stsAnnos := versionTemplate.Annotations
					if needIsomateName(stsAnnos) && !isomateNameArgs.DefaultIsomate {
						isomateSet.Spec.VersionTemplates[isomateResourceName] = versionTemplate
						delete(isomateSet.Spec.VersionTemplates, isoName)
					}
					isomateSetJsonMap, err := util.ConvertObjectToJsonMap(isomateSet)
					if err != nil {
						klog.Errorf("failed to convert isomate set to json map : %s", err.Error())
						return err
					}
					unstructured.Object = isomateSetJsonMap
				}

			} else {
				err := addNestedStringMap(unstructured.Object, map[string]string{k8s.IsomateNameLabelKey: isomateNameArgs.Name}, "metadata", "labels")
				if err != nil {
					klog.Errorf("failed to set isomate name label : %s", err.Error())
					return err
				}
				annos := unstructured.GetAnnotations()
				if needIsomateName(annos) {
					if !isomateNameArgs.DefaultIsomate {
						unstructured.SetName(isomateResourceName)
					}
				}
			}
		}
	}

	return nil
}

func needIsomateName(annos map[string]string) bool {
	return len(annos) > 0 && strings.ToLower(annos[NeedIsomateNameAnnoationKey]) == NeedIsomateNameAnnoationValue
}

func buildResourceName(unstructured *unstructured.Unstructured, isomateName string) string {
	return unstructured.GetName() + "-" + isomateName
}
