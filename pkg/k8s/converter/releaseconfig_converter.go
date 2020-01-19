package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"transwarp/release-config/pkg/apis/transwarp/v1beta1"
	"WarpCloud/walm/pkg/models/common"
)

func ConvertReleaseConfigFromK8s(oriReleaseConfig *v1beta1.ReleaseConfig) (*k8s.ReleaseConfig, error) {
	if oriReleaseConfig == nil {
		return nil, nil
	}
	releaseConfig := oriReleaseConfig.DeepCopy()
	return &k8s.ReleaseConfig{
		Meta:                     k8s.NewMeta(k8s.ReleaseConfigKind, releaseConfig.Namespace, releaseConfig.Name, k8s.NewState("", "", "")),
		Labels:                   releaseConfig.Labels,
		OutputConfig:             releaseConfig.Spec.OutputConfig,
		ChartImage:               releaseConfig.Spec.ChartImage,
		ChartName:                releaseConfig.Spec.ChartName,
		ConfigValues:             releaseConfig.Spec.ConfigValues,
		Dependencies:             releaseConfig.Spec.Dependencies,
		ChartVersion:             releaseConfig.Spec.ChartVersion,
		ChartAppVersion:          releaseConfig.Spec.ChartAppVersion,
		CreationTimestamp:        releaseConfig.CreationTimestamp.String(),
		Repo:                     releaseConfig.Spec.Repo,
		DependenciesConfigValues: releaseConfig.Spec.DependenciesConfigValues,
		IsomateConfig:            convertIsomateConfigFromK8s(releaseConfig.Spec.IsomateConfig),
		ChartWalmVersion:         common.WalmVersion(releaseConfig.Spec.ChartWalmVersion),
	}, nil
}

func convertIsomateConfigFromK8s(oriIsomateConfig *v1beta1.IsomateConfig) *k8s.IsomateConfig {
	if oriIsomateConfig == nil {
		return nil
	}
	isomateConfig := &k8s.IsomateConfig{
		DefaultIsomateName: oriIsomateConfig.DefaultIsomateName,
		Isomates:           []*k8s.Isomate{},
	}
	for _, oriIsomate := range oriIsomateConfig.Isomates {
		isomate := &k8s.Isomate{
			Name:         oriIsomate.Name,
			ConfigValues: oriIsomate.ConfigValues,
			Plugins:      []*k8s.ReleasePlugin{},
		}
		for _, plugin := range oriIsomate.Plugins {
			isomate.Plugins = append(isomate.Plugins, &k8s.ReleasePlugin{
				Name:    plugin.Name,
				Version: plugin.Version,
				Disable: plugin.Disable,
				Args:    plugin.Args,
			})
		}
		isomateConfig.Isomates = append(isomateConfig.Isomates, isomate)
	}
	return isomateConfig
}

func ConvertIsomateConfigToK8s(oriIsomateConfig *k8s.IsomateConfig) *v1beta1.IsomateConfig {
	if oriIsomateConfig == nil {
		return nil
	}
	isomateConfig := &v1beta1.IsomateConfig{
		DefaultIsomateName: oriIsomateConfig.DefaultIsomateName,
		Isomates:           []*v1beta1.Isomate{},
	}
	for _, oriIsomate := range oriIsomateConfig.Isomates {
		isomate := &v1beta1.Isomate{
			Name:         oriIsomate.Name,
			ConfigValues: oriIsomate.ConfigValues,
			Plugins:      []*v1beta1.ReleasePlugin{},
		}
		for _, plugin := range oriIsomate.Plugins {
			isomate.Plugins = append(isomate.Plugins, &v1beta1.ReleasePlugin{
				Name:    plugin.Name,
				Version: plugin.Version,
				Disable: plugin.Disable,
				Args:    plugin.Args,
			})
		}
		isomateConfig.Isomates = append(isomateConfig.Isomates, isomate)
	}
	return isomateConfig
}