package helm

import (
	"WarpCloud/walm/pkg/models/common"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release/utils"
	"encoding/json"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/tidwall/gjson"
	"k8s.io/klog"
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

func (helm *Helm) DryRunUpdateRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) (map[string]interface{}, error) {
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

	oldcmResources, err := getConfigMapResources(oldresources)
	if err != nil {
		return nil, err
	}
	cmResources, err := getConfigMapResources(resources)
	if err != nil {
		return nil, err
	}

	configmapList := []interface{}{}
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
			configmapList = append(configmapList, resource)
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

	dependedReleases := []map[string]string{}
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
					dependedReleases = append(dependedReleases, map[string]string{
						"name": releaseConfig.Name,
						"namespace": releaseConfig.Namespace,
						"chartName": releaseConfig.ChartName,
						"chartVersion": releaseConfig.ChartVersion,
						"repo": releaseConfig.Repo,
					})
				}
			}
		}
	}

	return map[string]interface{}{
		"dependedReleases": dependedReleases,
		"configmaps": configmapList,
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
