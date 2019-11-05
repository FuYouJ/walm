package plugins

import (
	"encoding/json"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
)

const (
	NodeSelectorPluginName = "NodeSelector"
)

func init() {
	register(NodeSelectorPluginName, &WalmPluginRunner{
		Run:  NodeSelectorTransform,
		Type: Pre_Install,
	})
}

type NodeSelectorRequirement struct {
	Key      string                  `json:"key"`
	Operator v1.NodeSelectorOperator `json:"operator"`
	Values   []string                `json:"values"`
}

type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions"`
	MatchFields      []NodeSelectorRequirement `json:"matchFields"`
}

type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm `json:"nodeSelectorTerms"`
}

type PreferredSchedulingTerm struct {
	Weight     int32            `json:"weight"`
	Preference NodeSelectorTerm `json:"preference"`
}

type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  NodeSelector              `json:"requiredDuringSchedulingIgnoredDuringExecution"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution"`
}

type Toleration struct {
	Key               string
	Operator          v1.TolerationOperator
	Value             string
	Effect            v1.TaintEffect
	TolerationSeconds *int64
}

type NodeSelectorArgs struct {
	NodeAffinity    NodeAffinity `json:"nodeAffinity"`
	NodeTolerations []Toleration `json:"nodeTolerations"`
}

func NodeSelectorTransform(context *PluginContext, args string) (err error) {
	if args == "" {
		klog.Infof("ignore node selector, because plugin args is empty")
		return nil
	} else {
		klog.Infof("node selector args : %s", args)
	}

	nodeSelectorArgs := &NodeSelectorArgs{}
	err = json.Unmarshal([]byte(args), nodeSelectorArgs)
	if err != nil {
		klog.Infof("failed to unmarshal plugin args : %s", err.Error())
		return err
	}

	preferedAffinity, requredAffinity := convertToNodeAffinity(nodeSelectorArgs.NodeAffinity)
	nodeTolerations := convertToToleration(nodeSelectorArgs.NodeTolerations)

	for _, resource := range context.Resources {
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Job", "Deployment", "DaemonSet", "StatefulSet":
			unStruct := resource.(*unstructured.Unstructured)
			err := mergeNodeToleration(unStruct, nodeTolerations)
			if err != nil {
				klog.Errorf("failed to add node toleration to pod : %s", err.Error())
				return err
			}
			err = mergeNodeAffinity(unStruct, preferedAffinity, requredAffinity)
			if err != nil {
				klog.Errorf("failed to add labels to pod : %s", err.Error())
				return err
			}
		}
	}

	return
}

func convertToNodeAffinity(nodeAffinityArgs NodeAffinity) ([]v1.PreferredSchedulingTerm, *v1.NodeSelector) {
	preferredNodeAffinity := make([]v1.PreferredSchedulingTerm, 0)
	requiredNodeAffinity := &v1.NodeSelector{}

	if len(nodeAffinityArgs.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		for _, preferredDuringSchedulingIgnoredDuringExecution := range nodeAffinityArgs.PreferredDuringSchedulingIgnoredDuringExecution {
			preferredSchedulingTerm := v1.PreferredSchedulingTerm{
				Weight: preferredDuringSchedulingIgnoredDuringExecution.Weight,
				Preference: v1.NodeSelectorTerm{
					MatchExpressions: make([]v1.NodeSelectorRequirement, 0),
				},
			}
			for _, preferenceMatchExpressions := range preferredDuringSchedulingIgnoredDuringExecution.Preference.MatchExpressions {
				preferredSchedulingTerm.Preference.MatchExpressions = append(preferredSchedulingTerm.Preference.MatchExpressions, v1.NodeSelectorRequirement{
					Key:      preferenceMatchExpressions.Key,
					Operator: preferenceMatchExpressions.Operator,
					Values:   preferenceMatchExpressions.Values,
				})
			}
			preferredNodeAffinity = append(preferredNodeAffinity, preferredSchedulingTerm)
		}
	}
	if len(nodeAffinityArgs.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
		requiredNodeAffinity = &v1.NodeSelector{
			NodeSelectorTerms: make([]v1.NodeSelectorTerm, 0),
		}
		for _, nodeSelectorTerm := range nodeAffinityArgs.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
			coreNodeSelectorTerm := v1.NodeSelectorTerm{
				MatchExpressions: make([]v1.NodeSelectorRequirement, 0),
			}
			for _, requireMatchExpressions := range nodeSelectorTerm.MatchExpressions {
				coreNodeSelectorTerm.MatchExpressions = append(coreNodeSelectorTerm.MatchExpressions, v1.NodeSelectorRequirement{
					Key:      requireMatchExpressions.Key,
					Operator: requireMatchExpressions.Operator,
					Values:   requireMatchExpressions.Values,
				})
			}
			requiredNodeAffinity.NodeSelectorTerms = append(requiredNodeAffinity.NodeSelectorTerms, coreNodeSelectorTerm)
		}
	}

	return preferredNodeAffinity, requiredNodeAffinity
}

func convertToToleration(tolerationsArgs []Toleration) []v1.Toleration {
	coreTolerations := make([]v1.Toleration, 0)

	for _, tolerationArgs := range tolerationsArgs {
		coreTolerations = append(coreTolerations, v1.Toleration{
			Key:      tolerationArgs.Key,
			Operator: tolerationArgs.Operator,
			Value:    tolerationArgs.Value,
			Effect:   tolerationArgs.Effect,
		})
	}

	return coreTolerations
}

func mergeNodeAffinity(unstructuredObj *unstructured.Unstructured, preferredNodeAffinity []v1.PreferredSchedulingTerm, requiredNodeAffinity *v1.NodeSelector) error {
	if len(preferredNodeAffinity) > 0 {
		preferredSchedulingTermsInterface := []interface{}{}
		for _, preferredSchedulingTerm := range preferredNodeAffinity {
			preferredSchedulingTermsInterface = append(preferredSchedulingTermsInterface, preferredSchedulingTerm)
		}
		err := addNestedSliceObj(
			unstructuredObj.Object, preferredSchedulingTermsInterface,
			"spec", "template", "spec", "affinity", "nodeAffinity", "preferredDuringSchedulingIgnoredDuringExecution",
		)
		if err != nil {
			klog.Errorf("failed to add nested slice obj : %s", err.Error())
			return err
		}
	}
	if requiredNodeAffinity != nil {
		nodeSelectorTermsInterface := []interface{}{}
		for _, nodeSelectorTerm := range requiredNodeAffinity.NodeSelectorTerms {
			nodeSelectorTermsInterface = append(nodeSelectorTermsInterface, nodeSelectorTerm)
		}
		err := addNestedSliceObj(
			unstructuredObj.Object, nodeSelectorTermsInterface,
			"spec", "template", "spec", "affinity", "nodeAffinity", "requiredDuringSchedulingIgnoredDuringExecution", "nodeSelectorTerms",
		)
		if err != nil {
			klog.Errorf("failed to add nested slice obj : %s", err.Error())
			return err
		}
	}

	return nil
}

func mergeNodeToleration(unstructuredObj *unstructured.Unstructured, nodeTolerations []v1.Toleration) error {
	tolerationsInterface := []interface{}{}
	if len(nodeTolerations) == 0 {
		return nil
	}
	for _, nodeToleration := range nodeTolerations {
		tolerationsInterface = append(tolerationsInterface, nodeToleration)
	}

	err := addNestedSliceObj(unstructuredObj.Object, tolerationsInterface, "spec", "template", "spec", "tolerations")
	if err != nil {
		klog.Errorf("failed to add nested slice obj : %s", err.Error())
		return err
	}

	return nil
}
