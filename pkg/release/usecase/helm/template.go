package helm

import (
	"WarpCloud/walm/pkg/models/common"
	"WarpCloud/walm/pkg/models/release"
	"k8s.io/klog"
)

func (helm *Helm) DryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) ([]map[string]interface{}, error) {
	releaseCache, err := helm.doInstallUpgradeRelease(namespace, releaseRequest, chartFiles, true, nil)
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

func (helm *Helm) ComputeResourcesByDryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, chartFiles []*common.BufferedFile) (*release.ReleaseResources, error) {
	r, err := helm.doInstallUpgradeRelease(namespace, releaseRequest, chartFiles, true, nil)
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
