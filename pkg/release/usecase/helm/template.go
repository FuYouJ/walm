package helm

import (
	"WarpCloud/walm/pkg/models/common"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release/utils"
	"encoding/json"
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

func (helm *Helm) DryRunUpdateRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) ([]*k8sModel.ReleaseConfig, error) {
	releaseCache, err := helm.doInstallUpgradeRelease(namespace, releaseRequest, nil, true)
	if err != nil {
		klog.Errorf("failed to dry run install release : %s", err.Error())
		return nil, err
	}
	resources, err := helm.k8sOperator.BuildManifestObjects(namespace, releaseCache.Manifest)
	if err != nil {
		klog.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}
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

	oldReleaseInfo, err := helm.GetRelease(releaseCache.Namespace, releaseCache.Name)
	if err != nil {
		return nil, err
	}
	var dependedReleases []*k8sModel.ReleaseConfig
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
					dependedReleases = append(dependedReleases, releaseConfig)
				}
			}
		}
	}
	return dependedReleases, nil
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
