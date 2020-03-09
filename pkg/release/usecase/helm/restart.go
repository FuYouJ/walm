package helm

import (
	"WarpCloud/walm/pkg/models/k8s"
	"fmt"
	"k8s.io/klog"
	"sync"
)

func (helm *Helm) RestartRelease(namespace, releaseName string) error {
	return helm.RestartReleaseIsomate(namespace, releaseName, "")
}

func (helm *Helm) RestartReleaseIsomate(namespace, releaseName, isomateName string) error {
	klog.V(2).Infof("Enter RestartRelease %s %s\n", namespace, releaseName)
	releaseInfo, err := helm.GetRelease(namespace, releaseName)
	if err != nil {
		klog.Errorf("failed to get release info : %s", err.Error())
		return err
	}

	var podsToRestart []*k8s.Pod
	if isomateName != "" {
		podsToRestart = releaseInfo.Status.GetIsomatePodsNeedRestart(isomateName)
	} else {
		podsToRestart = releaseInfo.Status.GetPodsNeedRestart()
	}

	podsRestartFailed := []string{}
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	for _, podToRestart := range podsToRestart {
		wg.Add(1)
		go func(podToRestart *k8s.Pod) {
			defer wg.Done()
			err1 := helm.k8sOperator.DeletePod(podToRestart.Namespace, podToRestart.Name)
			if err1 != nil {
				klog.Errorf("failed to restart pod %s/%s : %s", podToRestart.Namespace, podToRestart.Name, err1.Error())
				mux.Lock()
				podsRestartFailed = append(podsRestartFailed, podToRestart.Namespace+"/"+podToRestart.Name)
				mux.Unlock()
				return
			}
		}(podToRestart)
	}

	wg.Wait()
	if len(podsRestartFailed) > 0 {
		err = fmt.Errorf("failed to restart pods : %v", podsRestartFailed)
		klog.Errorf("failed to restart release : %s", err.Error())
		return err
	}

	klog.Infof("succeed to restart release %s", releaseName)
	return nil
}