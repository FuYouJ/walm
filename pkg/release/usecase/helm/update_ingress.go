package helm

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	"k8s.io/klog"
)

func (helm *Helm) UpdateReleaseIngress(
	namespace, name, ingressName string,
	requestBody *k8s.IngressRequestBody,
) error {
	releaseInfo, err := helm.GetRelease(namespace, name)
	if err != nil {
		klog.Errorf("failed to get release %s/%s : %s", namespace, name, err.Error())
		return err
	}

	found := false
	for _, releaseIngress := range releaseInfo.Status.Ingresses {
		if releaseIngress.Name == ingressName {
			found = true
			break
		}
	}
	if !found {
		return errorModel.NotFoundError{}
	}

	err = helm.k8sOperator.UpdateIngress(namespace, ingressName, requestBody)
	if err != nil {
		klog.Errorf("failed to update ingress release %s : %s", name, err.Error())
		return err
	}

	return nil
}
