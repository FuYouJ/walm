package helm

import (
	"github.com/hashicorp/golang-lru"
	"helm.sh/helm/pkg/kube"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	instanceclientset "transwarp/application-instance/pkg/client/clientset/versioned"
	"k8s.io/klog"
)

type Client struct {
	kubeWrappers *lru.Cache
	kubeConfig   string
	context      string
	k8sInstanceClient *instanceclientset.Clientset
}

type KubeWrapper struct {
	kubeConfig genericclioptions.RESTClientGetter
	kubeClient *kube.Client
}

func (c *Client) GetKubeClient(namespace string) (genericclioptions.RESTClientGetter, *kube.Client) {
	if c.kubeWrappers == nil {
		c.kubeWrappers, _ = lru.New(100)
	}

	if kubeWrapper, ok := c.kubeWrappers.Get(namespace); ok {
		kw := kubeWrapper.(KubeWrapper)
		return kw.kubeConfig, kw.kubeClient
	} else {
		kubeConfig := kube.GetConfig(c.kubeConfig, c.context, namespace)
		kubeClient := kube.New(kubeConfig, c.k8sInstanceClient)
		kubeClient.Log = klog.Infof
		c.kubeWrappers.Add(namespace, KubeWrapper{kubeConfig: kubeConfig, kubeClient: kubeClient})
		return kubeConfig, kubeClient
	}
}

func NewHelmKubeClient(kubeConfig string, context string, k8sInstanceClient *instanceclientset.Clientset) *Client {
	kubeClients, _ := lru.New(100)
	return &Client{
		kubeWrappers: kubeClients,
		kubeConfig:   kubeConfig,
		context:      context,
		k8sInstanceClient: k8sInstanceClient,
	}
}
