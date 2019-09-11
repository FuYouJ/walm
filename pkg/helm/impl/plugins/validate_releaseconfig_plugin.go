package plugins

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"transwarp/release-config/pkg/apis/transwarp/v1beta1"
)

const (
	AutoGenLabelKey = "auto-gen"
	ValidateReleaseConfigPluginName = "ValidateReleaseConfig"
)

// ValidateReleaseConfig plugin is used to make sure:
// 1. release have and only have one ReleaseConfig
// 2. ReleaseConfig has the same namespace and name with the release

func init() {
	register(ValidateReleaseConfigPluginName, &WalmPluginRunner{
		Run:  ValidateReleaseConfig,
		Type: Pre_Install,
	})
}

func ValidateReleaseConfig(context *PluginContext, args string) error {
	var autoGenReleaseConfig, releaseConfig *v1beta1.ReleaseConfig
	newResource := []runtime.Object{}
	for _, resource := range context.Resources {
		if resource.GetObjectKind().GroupVersionKind().Kind == "ReleaseConfig" {
			rc, err := buildReleaseConfig(resource.(*unstructured.Unstructured))
			if err != nil {
				klog.Infof("failed to convert unstructured : %s", err.Error())
				return err
			}
			if rc.Name != context.R.Name {
				continue
			}
			if len(rc.Labels) > 0 && rc.Labels[AutoGenLabelKey] == "true" {
				if autoGenReleaseConfig != nil {
					return fmt.Errorf("release can not have more than one auto-gen ReleaseConfig resource")
				} 
				autoGenReleaseConfig = rc
			} else {
				if releaseConfig != nil {
					return fmt.Errorf("release can not have more than one defined ReleaseConfig resource")
				}
				releaseConfig = rc
			}
		} else {
			newResource = append(newResource, resource)
		}
	}

	if autoGenReleaseConfig == nil {
		if releaseConfig == nil {
			return fmt.Errorf("release must have one ReleaseConfig resource")
		} else {
			newResource = append(newResource, releaseConfig)
		}
	} else {
		if len(autoGenReleaseConfig.Labels) > 0 {
			delete(autoGenReleaseConfig.Labels, AutoGenLabelKey)
		}
		if releaseConfig == nil {
			newResource = append(newResource, autoGenReleaseConfig)
		}else {
			autoGenReleaseConfig.Spec.OutputConfig = releaseConfig.Spec.OutputConfig
			if autoGenReleaseConfig.Labels == nil {
				autoGenReleaseConfig.Labels = map[string]string{}
			}

			for k, v := range releaseConfig.Labels {
				if _, ok := autoGenReleaseConfig.Labels[k]; !ok {
					autoGenReleaseConfig.Labels[k] = v
				}
			}

			newResource = append(newResource, autoGenReleaseConfig)
		}
	}

	context.Resources = newResource
	return nil
}

func buildReleaseConfig(resource *unstructured.Unstructured) (*v1beta1.ReleaseConfig, error) {
	releaseConfig := &v1beta1.ReleaseConfig{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal releaseConfig %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, releaseConfig)
	if err != nil {
		klog.Errorf("failed to unmarshal releaseConfig %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	return releaseConfig, nil
}

