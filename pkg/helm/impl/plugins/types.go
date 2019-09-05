package plugins

import (
	"helm.sh/helm/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime"
	"helm.sh/helm/pkg/release"
	releaseModel "WarpCloud/walm/pkg/models/release"
)

type RunnerType string

const (
	Pre_Install  RunnerType = "pre_install"
	Post_Install RunnerType = "post_install"
	Unknown      RunnerType = "unknown"

	WalmPluginConfigKey string = "Walm-Plugin-Key"
)

var pluginRunners map[string]*WalmPluginRunner

func register(name string, runner *WalmPluginRunner) {
	if pluginRunners == nil {
		pluginRunners = map[string]*WalmPluginRunner{}
	}
	pluginRunners[name] = runner
}

type WalmPluginRunner struct {
	Run  func(context *PluginContext, args string) error
	Type RunnerType
}

func GetRunner(walmPlugin *releaseModel.ReleasePlugin) *WalmPluginRunner {
	if pluginRunners == nil {
		return nil
	}
	return pluginRunners[walmPlugin.Name]
}

type PluginContext struct {
	KubeClient kube.Interface
	Resources  []runtime.Object
	R          *release.Release
}
