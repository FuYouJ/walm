package client

import (
	"walm/pkg/setting"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientsetex "transwarp/application-instance/pkg/client/clientset/versioned"
	"k8s.io/helm/pkg/kube"
	"github.com/sirupsen/logrus"
	releaseconfigclientset "transwarp/release-config/pkg/client/clientset/versioned"
)

var defaultApiserverClient *kubernetes.Clientset
var defaultRestConfig *restclient.Config
var defaultApiserverClientEx *clientsetex.Clientset
var defaultKubeClient *kube.Client
var defaultReleaseConfigClient *releaseconfigclientset.Clientset

func GetDefaultClient() *kubernetes.Clientset {
	var err error
	if defaultApiserverClient == nil {
		defaultApiserverClient, err = createApiserverClient("", setting.Config.KubeConfig.Config)
	}
	if err != nil {
		logrus.Fatalf("create apiserver client failed:%v", err)
	}
	return defaultApiserverClient
}

func GetDefaultClientEx() *clientsetex.Clientset {
	if defaultApiserverClientEx == nil {
		var err error
		defaultApiserverClientEx, err = createApiserverClientEx("", setting.Config.KubeConfig.Config)
		if err != nil {
			logrus.Fatalf("create apiserver client failed:%v", err)
		}
	}

	return defaultApiserverClientEx
}

func GetDefaultReleaseConfigClient() *releaseconfigclientset.Clientset {
	if defaultReleaseConfigClient == nil {
		var err error
		defaultReleaseConfigClient, err = createReleaseConfigClient("", setting.Config.KubeConfig.Config)
		if err != nil {
			logrus.Fatalf("create release config client failed:%v", err)
		}
	}

	return defaultReleaseConfigClient
}

func GetDefaultRestConfig() *restclient.Config {
	var err error
	if defaultRestConfig == nil {
		defaultRestConfig, err = clientcmd.BuildConfigFromFlags("", setting.Config.KubeConfig.Config)
	}
	if err != nil {
		logrus.Fatalf("get default rest config= failed:%v", err)
	}
	return defaultRestConfig
}

func GetKubeClient() *kube.Client {

	if defaultKubeClient == nil {
		defaultKubeClient = createKubeClient("", setting.Config.KubeConfig.Config)
	}

	return defaultKubeClient
}
