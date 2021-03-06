package helm

import (
	"WarpCloud/walm/pkg/k8s/converter"
	"WarpCloud/walm/pkg/models/common"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release/utils"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"reflect"
	"transwarp/release-config/pkg/apis/transwarp/v1beta1"
)

func (helm *Helm) DryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) ([]map[string]interface{}, error) {
	releaseCache, err := helm.doInstallUpgradeRelease(namespace, releaseRequest, chartFiles, true)
	if err != nil {
		klog.Errorf("failed to dry run install release : %s", err.Error())
		return nil, err
	}
	klog.V(2).Infof("release manifest : %s", releaseCache.Manifest)
	resources, err := helm.k8sOperator.BuildManifestObjects(namespace, releaseCache.Manifest)
	if err != nil {
		klog.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}

	return resources, nil
}

func (helm *Helm) DryRunUpdateRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) (*release.ReleaseDryRunUpdateInfo, error) {
	oldReleaseCache, err := helm.releaseCache.GetReleaseCache(namespace, releaseRequest.Name)
	if err != nil {
		klog.Errorf("failed to get release cache of %s/%s : %s", namespace, releaseRequest.Name, err.Error())
		return nil, err
	}
	oldReleaseInfo, err := helm.buildReleaseInfoV2(oldReleaseCache)
	if err != nil {
		return nil, err
	}
	releaseCache, err := helm.doInstallUpgradeRelease(namespace, releaseRequest, nil, true)
	if err != nil {
		klog.Errorf("failed to dry run install release : %s", err.Error())
		return nil, err
	}

	oldresources, err := helm.k8sOperator.BuildManifestObjects(namespace, oldReleaseCache.Manifest)
	if err != nil {
		klog.Errorf("failed to build old manifest objects: %s", err.Error())
	}
	resources, err := helm.k8sOperator.BuildManifestObjects(namespace, releaseCache.Manifest)
	if err != nil {
		klog.Errorf("failed to build manifest objects : %s", err.Error())
		return nil, err
	}

	// ReleaseConfig Diff
	oldrsResource, err := getResourceConfigResource(oldresources)
	if err != nil {
		return nil, err
	}
	rsResource, err := getResourceConfigResource(resources)
	if err != nil {
		return nil, err
	}

	k8sReleaseConfig := &v1beta1.ReleaseConfig{}
	oldSpec := oldrsResource["spec"]
	newSpec := rsResource["spec"]
	if !reflect.DeepEqual(oldSpec, newSpec) {
		data, err := json.Marshal(rsResource)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, k8sReleaseConfig)
		if err != nil {
			return nil, err
		}
	} else {
		k8sReleaseConfig = nil
	}

	releaseConfig, _ := converter.ConvertReleaseConfigFromK8s(k8sReleaseConfig)
	// ConfigMaps diff

	oldcmResources, err := getConfigMapResources(oldresources)
	if err != nil {
		return nil, err
	}
	cmResources, err := getConfigMapResources(resources)
	if err != nil {
		return nil, err
	}

	configmapList := []*corev1.ConfigMap{}
	for name, resource := range cmResources {
		oldResource := oldcmResources[name]
		oldResourceByte, err := json.Marshal(oldResource)
		if err != nil {
			return nil, err
		}
		resourceByte, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		oldData := gjson.GetBytes(oldResourceByte, "data").String()
		data := gjson.GetBytes(resourceByte, "data").String()
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(oldData, data, false)
		if len(diffs) > 1 {
			data, err := json.Marshal(resource)
			if err != nil {
				klog.Errorf("failed to marshal configmap interface: %s", err.Error())
				return nil, err
			}
			configmap := corev1.ConfigMap{}
			err = json.Unmarshal(data, &configmap)
			if err != nil {
				return nil, err
			}
			//k8sConfigMap, err := converter.ConvertConfigMapFromK8s(&configmap)
			configmapList = append(configmapList, &configmap)
		}
	}
	// compare outputValues
	outputConfigValues := map[string]interface{}{}
	for _, resource := range resources {
		if resource["kind"] == "ReleaseConfig" {
			resourceData, err := json.Marshal(resource)
			if err != nil {
				klog.Errorf("failed to marshal k8s resource: %s", err.Error())
				return nil, err
			}
			outputConfigStr := gjson.Get(string(resourceData), "spec.outputConfig").String()
			err = json.Unmarshal([]byte(outputConfigStr), &outputConfigValues)
			if err != nil {
				klog.Errorf("failed to unmarshal outputConfig: %s", err.Error())
			}
			break
		}
	}

	dependedReleases := []release.DependedRelease{}
	if utils.ConfigValuesDiff(oldReleaseInfo.OutputConfigValues, outputConfigValues) {
		releaseConfigs, err := helm.k8sCache.ListReleaseConfigs("", "")
		if err != nil {
			klog.Errorf("failed to list releaseconfigs: %s", err.Error())
			return nil, err
		}
		for _, releaseConfig := range releaseConfigs {
			for _, dependedRelease := range releaseConfig.Dependencies {
				dependedReleaseNamespace, dependedReleaseName, err := utils.ParseDependedRelease(releaseConfig.Namespace, dependedRelease)
				if err != nil {
					continue
				}
				if dependedReleaseNamespace == releaseCache.Namespace && dependedReleaseName == releaseCache.Name {
					dependedReleases = append(dependedReleases, release.DependedRelease{
						Name:         releaseConfig.Name,
						RepoName:     releaseConfig.Repo,
						ChartName:    releaseConfig.ChartName,
						ChartVersion: releaseConfig.ChartVersion,
						Namespace:    releaseConfig.Namespace,
					})
				}
			}
		}
	}

	return &release.ReleaseDryRunUpdateInfo{
		Configmaps:       configmapList,
		DependedReleases: dependedReleases,
		ReleaseConfig: releaseConfig,
	}, nil
}

func getConfigMapResources(resources []map[string]interface{}) (map[string]interface{}, error) {
	cmResources := map[string]interface{}{}
	for _, resource := range resources {
		if resource["kind"] == "ConfigMap" {
			data, err := json.Marshal(resource)
			if err != nil {
				return nil, err
			}
			name := gjson.GetBytes(data, "metadata.name").String()
			cmResources[name] = resource
		}
	}
	return cmResources, nil
}

func getResourceConfigResource(resources []map[string]interface{}) (map[string]interface{}, error) {
	for _, resource := range resources {
		if resource["kind"] == "ReleaseConfig" {
			return resource, nil
		}
	}
	return nil, errors.Errorf("releaseConfig not found in resources build by manifest")
}

func (helm *Helm) ComputeResourcesByDryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) (*release.ReleaseResources, error) {
	r, err := helm.doInstallUpgradeRelease(namespace, releaseRequest, chartFiles, true)
	if err != nil {
		klog.Errorf("failed to dry run install release : %s", err.Error())
		return nil, err
	}
	klog.V(2).Infof("release manifest : %s", r.Manifest)
	resources, err := helm.k8sOperator.ComputeReleaseResourcesByManifest(namespace, r.Manifest)
	if err != nil {
		klog.Errorf("failed to compute release resources by manifest : %s", err.Error())
		return nil, err
	}
	return resources, nil
}

func (helm *Helm) ComputeResourcesByGetRelease(namespace string, name string) (*release.ReleaseResources, error) {
	r, err := helm.releaseCache.GetReleaseCache(namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("release cache %s is not found in redis", name)
			return nil, err
		}
		klog.Errorf("failed to get release cache %s : %s", name, err.Error())
		return nil, err
	}
	resources, err := helm.k8sOperator.ComputeReleaseResourcesByManifest(namespace, r.Manifest)
	if err != nil {
		klog.Errorf("failed to compute release resources by manifest : %s", err.Error())
		return nil, err
	}
	return resources, nil
}

