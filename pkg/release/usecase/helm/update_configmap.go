package helm

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	"k8s.io/klog"
)

func (helm *Helm) UpdateReleaseConfigMap(
	namespace, name, configMapName string,
	requestBody *k8s.ConfigMapRequestBody,
) error {
	releaseInfo, err := helm.GetRelease(namespace, name)
	if err != nil {
		klog.Errorf("failed to get release %s/%s : %s", namespace, name, err.Error())
		return err
	}

	found := false
	for _, releaseConfigMap := range releaseInfo.Status.ConfigMaps {
		if releaseConfigMap.Name == configMapName {
			found = true
			break
		}
	}
	if !found {
		return errorModel.NotFoundError{}
	}

	err = helm.k8sOperator.UpdateConfigMap(namespace, configMapName, requestBody)
	if err != nil {
		klog.Errorf("failed to update configmap %s %s/%s: %s", configMapName, namespace, name, err.Error())
		return err
	}

	return nil
}
