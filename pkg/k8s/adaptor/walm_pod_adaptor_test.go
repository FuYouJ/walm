package adaptor

import (
	"testing"
	"walm/pkg/k8s/client"
	"fmt"
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodAdaptor(t *testing.T) {
	client, err := client.CreateFakeApiserverClient("", "C:/kubernetes/0.5/kubeconfig")
	if err != nil {
		println(err.Error())
		return
	}
	pod, err := client.CoreV1().Pods("default").Get("pi-qxhss", v1.GetOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}

	e, err := json.Marshal(BuildWalmPodState(*pod))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(e))
}