package helm

import (
	"WarpCloud/walm/pkg/models/common"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release/utils"
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
	releaseCache, err := helm.doInstallUpgradeRelease(namespace, releaseRequest, chartFiles, true)
	if err != nil {
		klog.Errorf("failed to dry run install release : %s", err.Error())
		return nil, err
	}
	releaseInfo, err := helm.buildReleaseInfoV2(releaseCache)
	if err != nil {
		klog.Errorf("failed to build releaseInfo: %s", err.Error())
		return nil, err
	}

	oldReleaseInfo, err := helm.GetRelease(releaseCache.Namespace, releaseCache.Name)
	if err != nil {
		return nil, err
	}
	var dependedReleases []*k8sModel.ReleaseConfig
	if utils.ConfigValuesDiff(oldReleaseInfo.OutputConfigValues, releaseInfo.OutputConfigValues) {
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
				if dependedReleaseNamespace == releaseInfo.Namespace && dependedReleaseName == releaseInfo.Name {
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
