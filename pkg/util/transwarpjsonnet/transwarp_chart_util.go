package transwarpjsonnet

import (
	"encoding/json"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"helm.sh/helm/pkg/chart"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"transwarp/release-config/pkg/apis/transwarp/v1beta1"

	"WarpCloud/walm/pkg/helm/impl/plugins"
	"WarpCloud/walm/pkg/k8s/converter"
	"WarpCloud/walm/pkg/models/common"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/setting"
	"WarpCloud/walm/pkg/util"
)

const (
	CommonTemplateDir             = "applib/ksonnet-lib"
	TranswarpJsonetFileSuffix     = ".transwarp-jsonnet.yaml"
	TranswarpJsonnetTemplateDir   = "template-jsonnet/"
	TranswarpJsonetAppLibDir      = "applib/"
	TranswarpMetadataDir          = "transwarp-meta/"
	TranswarpCiDir                = "ci/"
	TranswarpMetaInfoFileName     = "metainfo.yaml"
	TranswarpIconFileName         = "icon"
	TranswarpAdvantageFileName    = "advantage.html"
	TranswarpArchitectureFileName = "architecture.html"
	TranswarpAppYamlPattern       = TranswarpJsonnetTemplateDir + "%s/%s/app.yaml"

	TranswarpInstallIDKey        = "Transwarp_Install_ID"
	TranswarpInstallNamespaceKey = "Transwarp_Install_Namespace"
)

var commonTemplateFilesPath string
var commonTemplateFiles map[string]string

// LoadFilesFromDisk loads all files inside baseDir directory and its subdirectory recursively,
// mapping each file's path/content as a key/value into a map.
func loadFilesFromDisk(baseDir string) (map[string]string, error) {
	cacheFiles := make(map[string]string)
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				klog.Errorf("Read file \"%s\", err: %v", path, err)
				return err
			}
			cacheFiles[strings.TrimPrefix(filepath.ToSlash(path), baseDir)] = string(b)
		}
		return nil
	})
	if err != nil {
		return cacheFiles, err
	}
	return cacheFiles, nil
}

func loadCommonJsonnetLib(templates map[string]string) (err error) {
	if commonTemplateFiles == nil {
		if len(commonTemplateFilesPath) == 0 && setting.Config.JsonnetConfig != nil {
			commonTemplateFilesPath = setting.Config.JsonnetConfig.CommonTemplateFilesPath
		}
		if commonTemplateFilesPath == "" {
			return
		}
		commonTemplateFiles, err = loadFilesFromDisk(commonTemplateFilesPath)
		if err != nil {
			klog.Errorf("failed to load common template files : %s", err.Error())
			return
		}
	}
	for key, value := range commonTemplateFiles {
		templates[path.Join(CommonTemplateDir, key)] = value
	}
	return nil
}

type HelmNativeValues struct {
	ChartName        string `json:"chartName"`
	ChartVersion     string `json:"chartVersion"`
	AppVersion       string `json:"appVersion"`
	ReleaseName      string `json:"releaseName"`
	ReleaseNamespace string `json:"releaseNamespace"`
}

type AppHelmValues struct {
	Dependencies map[string]string `json:"dependencies"`
	NativeValues HelmNativeValues  `json:"HelmNativeValues"`
}

func buildConfigValuesToRender(
	rawChart *chart.Chart, namespace, name string,
	userConfigs, dependencyConfigs map[string]interface{}, dependencies map[string]string,
	isomateConfigValue map[string]interface{}) (configValues map[string]interface{}, err error) {
	configValues = map[string]interface{}{}
	util.MergeValues(configValues, setting.Config.AdditionAppConfig.ExtraConfig, false)
	util.MergeValues(configValues, rawChart.Values, false)
	util.MergeValues(configValues, dependencyConfigs, false)

	//config TosVersion can be override by user configs
	configValues["TosVersion"] = setting.Config.AdditionAppConfig.TosVersion

	util.MergeValues(configValues, userConfigs, false)
	util.MergeValues(configValues, isomateConfigValue, false)
	// the configs below can not be override by user configs
	configValues["helmReleaseName"] = name
	configValues["helmReleaseNamespace"] = namespace
	configValues["chartVersion"] = rawChart.Metadata.Version
	configValues["chartName"] = rawChart.Metadata.Name
	configValues["chartAppVersion"] = rawChart.Metadata.AppVersion
	configValues[TranswarpInstallNamespaceKey] = namespace

	//Compatible
	configValues["Customized_Namespace"] = namespace

	helmVals := AppHelmValues{}
	helmVals.NativeValues.ChartName = rawChart.Metadata.Name
	helmVals.NativeValues.ChartVersion = rawChart.Metadata.Version
	helmVals.NativeValues.AppVersion = rawChart.Metadata.AppVersion
	helmVals.NativeValues.ReleaseName = name
	helmVals.NativeValues.ReleaseNamespace = namespace
	helmVals.Dependencies = dependencies

	chartRawBase := map[string]interface{}{}
	chartRawBase["HelmAdditionalValues"] = helmVals
	chartJsonRawBase := map[string]interface{}{}
	chartJsonRawVals, _ := yaml.Marshal(chartRawBase)
	yaml.Unmarshal(chartJsonRawVals, &chartJsonRawBase)
	util.MergeValues(configValues, chartJsonRawBase, false)

	return configValues, nil
}

func ProcessChart(chartInfo *release.ChartDetailInfo, releaseRequest *release.ReleaseRequestV2, rawChart *chart.Chart,
	namespace string, configValues, dependencyConfigs map[string]interface{}, dependencies, releaseLabels map[string]string,
	isomateConfigValue map[string]interface{}, updateConfigMap bool) (err error) {
	if chartInfo.WalmVersion == common.WalmVersionV2 {
		err = ProcessJsonnetChart(
			releaseRequest.RepoName, rawChart, namespace,
			releaseRequest.Name, configValues, dependencyConfigs,
			dependencies, releaseLabels, releaseRequest.ChartImage,
			releaseRequest.IsomateConfig, isomateConfigValue, common.WalmVersionV2, updateConfigMap)
		if err != nil {
			klog.Errorf("failed to ProcessJsonnetChart : %s", err.Error())
			return
		}
	} else if chartInfo.WalmVersion == common.WalmVersionV1 {
		err = ProcessJsonnetChartV1(
			releaseRequest.RepoName, rawChart, namespace,
			releaseRequest.Name, configValues, dependencyConfigs,
			dependencies, releaseLabels, releaseRequest.ChartImage,
			releaseRequest.IsomateConfig, isomateConfigValue, common.WalmVersionV1, updateConfigMap)
		if err != nil {
			klog.Errorf("failed to ProcessJsonnetChart v1: %s", err.Error())
			return
		}
	}
	return
}

// convert jsonnet chart to native chart
// 1. load jsonnet template files to render
//     a. load common jsonnet lib
//     b. load jsonnet chart template files
// 2. build config values to render jsonnet template files
//     a. merge values from value.yaml
//     b. merge system values
//     c. merge dependency release output configs
//     d. merge configs user provided
// 3. render jsonnet template files to generate native chart templates
func ProcessJsonnetChart(repo string, rawChart *chart.Chart, releaseNamespace,
	releaseName string, userConfigs, dependencyConfigs map[string]interface{},
	dependencies, releaseLabels map[string]string, chartImage string, isomateConfig *k8s.IsomateConfig,
	isomateConfigValue map[string]interface{}, chartWalmVersion common.WalmVersion, updateConfigMap bool) error {
	jsonnetTemplateFiles := make(map[string]string, 0)
	var rawChartFiles []*chart.File
	for _, f := range rawChart.Files {
		if strings.HasPrefix(f.Name, TranswarpJsonnetTemplateDir) {
			cname := strings.TrimPrefix(f.Name, TranswarpJsonnetTemplateDir)
			if strings.IndexAny(cname, "._") == 0 {
				// Ignore charts/ that start with . or _.
				continue
			}
			appcname := path.Join(releaseName, rawChart.Metadata.AppVersion, TranswarpJsonnetTemplateDir, cname)
			jsonnetTemplateFiles[appcname] = string(f.Data)
		} else if strings.HasPrefix(f.Name, TranswarpJsonetAppLibDir) {
			jsonnetTemplateFiles[f.Name] = string(f.Data)
		} else if !strings.HasPrefix(f.Name, TranswarpCiDir) {
			rawChartFiles = append(rawChartFiles, f)
		}
	}

	autoGenReleaseConfig, err := buildAutoGenReleaseConfig(
		releaseNamespace, releaseName, repo,
		rawChart.Metadata.Name, rawChart.Metadata.Version,
		rawChart.Metadata.AppVersion, releaseLabels, dependencies,
		dependencyConfigs, userConfigs, chartImage, isomateConfig, chartWalmVersion)
	if err != nil {
		klog.Errorf("failed to auto gen release config : %s", err.Error())
		return err
	}
	rawChart.Templates = append(rawChart.Templates, &chart.File{
		Name: BuildNotRenderedFileName("autogen-releaseconfig.json"),
		Data: autoGenReleaseConfig,
	})
	rawChart.Files = rawChartFiles

	if len(jsonnetTemplateFiles) == 0 {
		// native chart
		klog.Infof("chart %s is native chart", rawChart.Metadata.Name)
		return nil
	}

	// load values.yaml
	valueYamlContent, err := json.Marshal(rawChart.Values)
	jsonnetTemplateFiles[path.Join(releaseName, rawChart.Metadata.AppVersion, "values.yaml")] = string(valueYamlContent)

	loadCommonJsonnetLib(jsonnetTemplateFiles)

	configValues, err := buildConfigValuesToRender(rawChart, releaseNamespace, releaseName, userConfigs, dependencyConfigs, dependencies, isomateConfigValue)
	if err != nil {
		klog.Errorf("failed to build config values to render jsonnet template files : %s", err.Error())
		return err
	}
	jsonStr, err := renderMainJsonnetFile(jsonnetTemplateFiles, configValues)
	if err != nil {
		klog.Errorf("failed to render jsonnet files : %s", err.Error())
		return err
	}

	// adjust for kundb devops modules
	k8sLabels := map[string]string{}
	if releaseLabels["project-name"] != "" {
		k8sLabels["project-name"] = releaseLabels["project-name"]
	}
	if releaseLabels["product-line"] != "" {
		k8sLabels["product-line"] = releaseLabels["product-line"]
	}

	kubeResources, err := buildKubeResourcesByJsonStr(jsonStr, k8sLabels, updateConfigMap)
	if err != nil {
		klog.Errorf("failed to build native chart templates : %s", err.Error())
		return err
	}

	for fileName, kubeResourceBytes := range kubeResources {
		rawChart.Templates = append(rawChart.Templates, &chart.File{
			Name: BuildNotRenderedFileName(fileName),
			Data: kubeResourceBytes,
		})
	}
	return nil
}

func ProcessJsonnetChartV1(
	repo string, rawChart *chart.Chart, releaseNamespace,
	releaseName string, userConfigs, dependencyConfigs map[string]interface{},
	dependencies, releaseLabels map[string]string, chartImage string, isomateConfig *k8s.IsomateConfig,
	isomateConfigValue map[string]interface{}, chartWalmVersion common.WalmVersion, updateConfigMap bool) error {
	jsonnetTemplateFiles := make(map[string]string, 0)
	var rawChartFiles []*chart.File
	for _, f := range rawChart.Files {
		if strings.HasPrefix(f.Name, TranswarpJsonnetTemplateDir) {
			cname := strings.TrimPrefix(f.Name, TranswarpJsonnetTemplateDir)
			if strings.IndexAny(cname, "._") == 0 {
				// Ignore charts/ that start with . or _.
				continue
			}
			appcname := path.Join(releaseName, rawChart.Metadata.AppVersion, TranswarpJsonnetTemplateDir, cname)
			jsonnetTemplateFiles[appcname] = string(f.Data)
		} else {
			rawChartFiles = append(rawChartFiles, f)
		}
	}

	autoGenReleaseConfig, err := buildAutoGenReleaseConfig(
		releaseNamespace, releaseName, repo,
		rawChart.Metadata.Name, rawChart.Metadata.Version,
		rawChart.Metadata.AppVersion, releaseLabels, dependencies,
		dependencyConfigs, userConfigs, chartImage, isomateConfig, chartWalmVersion)
	if err != nil {
		klog.Errorf("failed to auto gen release config : %s", err.Error())
		return err
	}
	rawChart.Templates = append(rawChart.Templates, &chart.File{
		Name: BuildNotRenderedFileName("autogen-releaseconfig.json"),
		Data: autoGenReleaseConfig,
	})
	rawChart.Files = rawChartFiles

	if len(jsonnetTemplateFiles) == 0 {
		// native chart
		klog.Infof("chart %s is native chart", rawChart.Metadata.Name)
		return nil
	}

	configValues, err := buildConfigValuesToRender(rawChart, releaseNamespace, releaseName, userConfigs, dependencyConfigs, dependencies, isomateConfigValue)
	if err != nil {
		klog.Errorf("failed to build config values to render jsonnet template files : %s", err.Error())
		return err
	}
	jsonStr, err := renderMainJsonnetFile(jsonnetTemplateFiles, configValues)
	if err != nil {
		klog.Errorf("failed to render jsonnet files : %s", err.Error())
		return err
	}

	// adjust for kundb devops modules
	k8sLabels := map[string]string{}
	if releaseLabels["project-name"] != "" {
		k8sLabels["project-name"] = releaseLabels["project-name"]
	}
	if releaseLabels["product-line"] != "" {
		k8sLabels["product-line"] = releaseLabels["product-line"]
	}
	kubeResources, err := buildKubeResourcesByJsonStr(jsonStr, k8sLabels, updateConfigMap)
	if err != nil {
		klog.Errorf("failed to build native chart templates : %s", err.Error())
		return err
	}

	for fileName, kubeResourceBytes := range kubeResources {
		rawChart.Templates = append(rawChart.Templates, &chart.File{
			Name: BuildNotRenderedFileName(fileName),
			Data: kubeResourceBytes,
		})
	}
	return nil
}

func buildAutoGenReleaseConfig(releaseNamespace, releaseName, repo, chartName, chartVersion, chartAppVersion string,
	labels, dependencies map[string]string, dependencyConfigs, userConfigs map[string]interface{}, chartImage string,
	isomateConfig *k8s.IsomateConfig, chartWalmVersion common.WalmVersion) ([]byte, error) {
	if labels == nil {
		labels = map[string]string{}
	}
	labels[plugins.AutoGenLabelKey] = "true"

	releaseConfig := &v1beta1.ReleaseConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReleaseConfig",
			APIVersion: "apiextensions.transwarp.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: releaseNamespace,
			Name:      releaseName,
			Labels:    labels,
		},
		Spec: v1beta1.ReleaseConfigSpec{
			DependenciesConfigValues: dependencyConfigs,
			ChartVersion:             chartVersion,
			ChartName:                chartName,
			ChartAppVersion:          chartAppVersion,
			ConfigValues:             userConfigs,
			Dependencies:             dependencies,
			OutputConfig:             map[string]interface{}{},
			Repo:                     repo,
			ChartImage:               chartImage,
			IsomateConfig:            converter.ConvertIsomateConfigToK8s(isomateConfig),
			ChartWalmVersion:         string(chartWalmVersion),
		},
	}

	releaseConfigBytes, err := yaml.Marshal(releaseConfig)
	if err != nil {
		klog.Errorf("failed to marshal release config : %s", err.Error())
		return nil, err
	}
	return releaseConfigBytes, nil
}
