package adaptor

import (
	corev1 "k8s.io/api/core/v1"
	"walm/pkg/instance/walmlister"
)

type WalmPodAdaptor struct{
	Lister walmlister.K8sResourceLister
}

func (adaptor WalmPodAdaptor) GetWalmPods(namespace string, labelSelectorStr string) ([]WalmPod, error) {
	podList, err := adaptor.Lister.GetPods(namespace, labelSelectorStr)
	if err != nil {
		return nil, err
	}

	walmPods := []WalmPod{}
	if podList != nil {
		for _, pod := range podList.Items {
			walmPod := BuildWalmPod(pod)
			walmPods = append(walmPods, walmPod)
		}
	}

	return walmPods, nil
}

func BuildWalmPod(pod corev1.Pod) WalmPod {
	walmPod := WalmPod{
		WalmMeta: WalmMeta{pod.Name, pod.Namespace},
		PodIp:    pod.Status.PodIP,
	}
	return walmPod
}