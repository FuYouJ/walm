package framework

import (
	"strings"
	"fmt"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"errors"
	"WarpCloud/walm/pkg/setting"
	"github.com/sirupsen/logrus"
	"WarpCloud/walm/pkg/k8s/client"
	"k8s.io/client-go/kubernetes"
	clienthelm "WarpCloud/walm/pkg/k8s/client/helm"
	releaseconfigclientset "transwarp/release-config/pkg/client/clientset/versioned"
	"runtime"
	"WarpCloud/walm/pkg/helm/impl"
	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/registry"
)

var k8sClient *kubernetes.Clientset
var k8sReleaseConfigClient *releaseconfigclientset.Clientset
var kubeClients *clienthelm.Client

const (
	maxNameLength                = 62
	randomLength                 = 5
	maxGeneratedRandomNameLength = maxNameLength - randomLength

	// For helm test
	TestChartRepoName  = "test"
	TomcatChartName    = "tomcat"
	TomcatChartVersion = "0.2.0"

	V1ZookeeperChartName = "zookeeper"
	V1ZookeeperChartVersion = "5.2.0"

	tomcatChartImageSuffix = "walm-test/tomcat:0.2.0"
)

func GenerateRandomName(base string) string {
	if len(base) > maxGeneratedRandomNameLength {
		base = base[:maxGeneratedRandomNameLength]
	}
	return fmt.Sprintf("%s-%s", strings.ToLower(base), utilrand.String(randomLength))
}

func GetCurrentFilePath() (string, error) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("Can not get current file info")
	}
	return file, nil
}

func InitFramework() error {
	tomcatChartPath, err := GetLocalTomcatChartPath()
	if err != nil {
		logrus.Errorf("failed to get tomcat chart path : %s", err.Error())
		return err
	}

	v1ZookeeperChartPath, err := GetLocalV1ZookeeperChartPath()
	if err != nil {
		logrus.Errorf("failed to get v1 zookeeper chart path : %s", err.Error())
		return err
	}

	foundTestRepo := false
	for _, repo := range setting.Config.RepoList {
		if repo.Name == TestChartRepoName{
			foundTestRepo = true
			err = PushChartToRepo(repo.URL, tomcatChartPath)
			if err != nil {
				logrus.Errorf("failed to push tomcat chart to repo : %s", err.Error())
				return err
			}
			err = PushChartToRepo(repo.URL, v1ZookeeperChartPath)
			if err != nil {
				logrus.Errorf("failed to push v1 zookeeper chart to repo : %s", err.Error())
				return err
			}
			break
		}
	}
	if !foundTestRepo {
		return fmt.Errorf("repo %s is not found", TestChartRepoName)
	}

	if setting.Config.ChartImageRegistry == "" {
		return errors.New("chart image registry should not be empty")
	}

	chartImage := GetTomcatChartImage()
	logrus.Infof("start to push chart image %s to registry", chartImage)
	registryClient, err := impl.NewRegistryClient(setting.Config.ChartImageConfig)
	if err != nil {
		return err
	}

	testChart, err := loader.Load(tomcatChartPath)
	if err != nil {
		logrus.Errorf("failed to load test chart : %s", err.Error())
		return err
	}

	ref, err := registry.ParseReference(chartImage)
	if err != nil {
		logrus.Errorf("failed to parse chart image %s : %s", chartImage, err.Error())
		return err
	}

	registryClient.SaveChart(testChart, ref)
	err = registryClient.PushChart(ref)
	if err != nil {
		logrus.Errorf("failed to push chart image : %s", err.Error())
		return err
	}

	kubeConfig := ""
	if setting.Config.KubeConfig != nil {
		kubeConfig = setting.Config.KubeConfig.Config
	}
	kubeContext := ""
	if setting.Config.KubeConfig != nil {
		kubeContext = setting.Config.KubeConfig.Context
	}

	k8sClient, err = client.NewClient("", kubeConfig)
	if err != nil {
		logrus.Errorf("failed to create k8s client : %s", err.Error())
		return err
	}

	k8sReleaseConfigClient, err = client.NewReleaseConfigClient("", kubeConfig)
	if err != nil {
		logrus.Errorf("failed to create k8s release config client : %s", err.Error())
		return err
	}

	kubeClients = clienthelm.NewHelmKubeClient(kubeConfig, kubeContext)

	return nil
}

