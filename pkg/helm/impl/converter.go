package impl

import (
	"WarpCloud/walm/pkg/models/release"
	"github.com/tidwall/gjson"
	"WarpCloud/walm/pkg/k8s/utils"
)

const (
	metaRoleBaseConfigReplicasName       = "replicas"
	metaRoleBaseConfigUseHostNetworkName = "useHostNetwork"
	metaRoleBaseConfigPriorityName       = "priority"
	metaRoleBaseConfigEnvName            = "envList"
	metaRoleBaseConfigEnvMapName         = "envMap"
	metaRoleBaseConfigImageName          = "image"
)

func convertUserInputParams(userInputParams *UserInputParams) *release.PrettyChartParams {
	if userInputParams == nil {
		return nil
	}
	prettyChartParams := &release.PrettyChartParams{}

	for _, role := range userInputParams.CommonConfig.Roles {
		prettyChartParams.CommonConfig.Roles = append(prettyChartParams.CommonConfig.Roles, convertRoleConfig(role))
	}

	for _, baseConfig := range userInputParams.AdvanceConfig {
		prettyChartParams.AdvanceConfig = append(prettyChartParams.AdvanceConfig, convertBaseConfig(baseConfig))
	}

	for _, baseConfig := range userInputParams.TranswarpBaseConfig {
		prettyChartParams.TranswarpBaseConfig = append(prettyChartParams.TranswarpBaseConfig, convertBaseConfig(baseConfig))
	}
	return prettyChartParams
}

func convertBaseConfig(baseConfig *BaseConfig) *release.BaseConfig {
	if baseConfig == nil {
		return nil
	}
	return &release.BaseConfig{
		Variable:         baseConfig.ValueName,
		DefaultValue:     baseConfig.DefaultValue,
		ValueDescription: baseConfig.ValueDescription,
		ValueType:        baseConfig.ValueType,
	}
}

func convertRoleConfig(roleConfig *RoleConfig) *release.RoleConfig {
	if roleConfig == nil {
		return nil
	}
	rc := &release.RoleConfig{
		Name:               roleConfig.Name,
		Description:        roleConfig.Description,
		Replicas:           roleConfig.Replicas,
		RoleResourceConfig: convertResourceConfig(roleConfig.RoleResourceConfig),
	}

	for _, roleBaseConfig := range roleConfig.RoleBaseConfig {
		rc.RoleBaseConfig = append(rc.RoleBaseConfig, convertBaseConfig(roleBaseConfig))
	}
	return rc
}

func convertResourceConfig(resourceConfig *ResourceConfig) *release.ResourceConfig {
	if resourceConfig == nil {
		return nil
	}
	rc := &release.ResourceConfig{
		MemoryRequest: resourceConfig.MemoryRequest,
		MemoryLimit:   resourceConfig.MemoryLimit,
		CpuRequest:    resourceConfig.CpuRequest,
		CpuLimit:      resourceConfig.CpuLimit,
		GpuLimit:      resourceConfig.GpuLimit,
		GpuRequest:    resourceConfig.GpuRequest,
	}

	for _, storageConfig := range resourceConfig.ResourceStorageList {
		rc.ResourceStorageList = append(rc.ResourceStorageList, convertResourceStorageConfig(storageConfig))
	}
	return rc
}

func convertResourceStorageConfig(resourceStorageConfig ResourceStorageConfig) release.ResourceStorageConfig {
	return release.ResourceStorageConfig{
		Name:         resourceStorageConfig.Name,
		DiskReplicas: resourceStorageConfig.DiskReplicas,
		StorageType:  resourceStorageConfig.StorageType,
		AccessModes:  resourceStorageConfig.AccessModes,
		Size:         resourceStorageConfig.Size,
		StorageClass: resourceStorageConfig.StorageClass,
		AccessMode:   resourceStorageConfig.AccessMode,
	}
}

func convertPrettyParamsToMetainfoParams(prettyParams *release.PrettyChartParams) *release.MetaInfoParams {
	return nil
}

func convertMetainfoToPrettyParams(metaInfo *release.ChartMetaInfo) (*release.PrettyChartParams) {
	if metaInfo == nil || (len(metaInfo.ChartParams) == 0 && len(metaInfo.ChartRoles) == 0) {
		return nil
	}

	prettyParams := &release.PrettyChartParams{}
	for _, chartParam := range metaInfo.ChartParams {
		baseConfig, variableType := convertChartParamToBaseConfig(chartParam)
		switch variableType {
		case release.AdvanceConfig:
			prettyParams.AdvanceConfig = append(prettyParams.AdvanceConfig, baseConfig)
		case release.TranswarpBundleConfig:
			prettyParams.TranswarpBaseConfig = append(prettyParams.TranswarpBaseConfig, baseConfig)
		default:
			prettyParams.AdvanceConfig = append(prettyParams.AdvanceConfig, baseConfig)
		}
	}

	for _, chartRole := range metaInfo.ChartRoles {
		roleConfig := convertChartRoleToRoleConfig(chartRole)
		prettyParams.CommonConfig.Roles = append(prettyParams.CommonConfig.Roles, roleConfig)
	}

	return prettyParams
}

func convertChartRoleToRoleConfig(metaRoleConfig *release.MetaRoleConfig) (roleConfig *release.RoleConfig) {
	if metaRoleConfig != nil {
		roleConfig = &release.RoleConfig{
			Name:           metaRoleConfig.Name,
			Description:    metaRoleConfig.Description,
			RoleBaseConfig: convertMetaRoleBaseConfigToBaseConfigs(metaRoleConfig.RoleBaseConfig),
			RoleResourceConfig: convertMetaResourceConfigToResourceConfig(metaRoleConfig.RoleResourceConfig),
		}
	}
	return
}

func convertMetaResourceConfigToResourceConfig(metaResourceConfig *release.MetaResourceConfig) (resourceConfig *release.ResourceConfig) {
	if metaResourceConfig != nil {
		resourceConfig = &release.ResourceConfig{}
		if metaResourceConfig.RequestsCpu != nil {
			resourceConfig.CpuRequest = metaResourceConfig.RequestsCpu.DefaultValue
		}
		if metaResourceConfig.LimitsCpu != nil {
			resourceConfig.CpuLimit = metaResourceConfig.LimitsCpu.DefaultValue
		}
		if metaResourceConfig.RequestsGpu != nil {
			resourceConfig.GpuRequest = int(metaResourceConfig.RequestsGpu.DefaultValue)
		}
		if metaResourceConfig.LimitsGpu != nil {
			resourceConfig.GpuLimit = int(metaResourceConfig.LimitsGpu.DefaultValue)
		}
		if metaResourceConfig.RequestsMemory != nil {
			resourceConfig.MemoryRequest = float64(metaResourceConfig.RequestsMemory.DefaultValue)
		}
		if metaResourceConfig.LimitsMemory != nil {
			resourceConfig.MemoryLimit = float64(metaResourceConfig.LimitsMemory.DefaultValue)
		}
		for _, metaResourceConfig := range metaResourceConfig.StorageResources {
			resourceConfig.ResourceStorageList = append(resourceConfig.ResourceStorageList, convertMetaResourceStorageConfig(metaResourceConfig))
		}
	}
	return
}

func convertMetaResourceStorageConfig(metaResourceStorageConfig *release.MetaResourceStorageConfig) (resourceStorageConfig release.ResourceStorageConfig) {
	if metaResourceStorageConfig != nil {
		resourceStorageConfig = release.ResourceStorageConfig{
			Name: metaResourceStorageConfig.Name,
		}
		if metaResourceStorageConfig.DefaultValue != nil {
			resourceStorageConfig.StorageClass = metaResourceStorageConfig.DefaultValue.StorageClass
			resourceStorageConfig.AccessModes = metaResourceStorageConfig.DefaultValue.AccessModes
			resourceStorageConfig.DiskReplicas = metaResourceStorageConfig.DefaultValue.DiskReplicas
			resourceStorageConfig.Size = release.ConvertResourceBinaryIntByUnit(&metaResourceStorageConfig.DefaultValue.Size, utils.K8sResourceStorageUnit)
		}
	}
	return
}

func convertMetaRoleBaseConfigToBaseConfigs(metaRoleBaseConfig *release.MetaRoleBaseConfig) (baseConfigs []*release.BaseConfig) {
	if metaRoleBaseConfig != nil {
		if metaRoleBaseConfig.Replicas != nil {
			baseConfigs = append(baseConfigs, &release.BaseConfig{
				Name:             metaRoleBaseConfigReplicasName,
				Variable:         metaRoleBaseConfig.Replicas.Variable,
				ValueType:        metaRoleBaseConfig.Replicas.Type,
				ValueDescription: metaRoleBaseConfig.Replicas.Description,
				DefaultValue:     metaRoleBaseConfig.Replicas.DefaultValue,
			})
		}
		if metaRoleBaseConfig.UseHostNetwork != nil {
			baseConfigs = append(baseConfigs, &release.BaseConfig{
				Name:             metaRoleBaseConfigUseHostNetworkName,
				Variable:         metaRoleBaseConfig.UseHostNetwork.Variable,
				ValueType:        metaRoleBaseConfig.UseHostNetwork.Type,
				ValueDescription: metaRoleBaseConfig.UseHostNetwork.Description,
				DefaultValue:     metaRoleBaseConfig.UseHostNetwork.DefaultValue,
			})
		}
		if metaRoleBaseConfig.Priority != nil {
			baseConfigs = append(baseConfigs, &release.BaseConfig{
				Name:             metaRoleBaseConfigPriorityName,
				Variable:         metaRoleBaseConfig.Priority.Variable,
				ValueType:        metaRoleBaseConfig.Priority.Type,
				ValueDescription: metaRoleBaseConfig.Priority.Description,
				DefaultValue:     metaRoleBaseConfig.Priority.DefaultValue,
			})
		}
		if metaRoleBaseConfig.Env != nil {
			baseConfigs = append(baseConfigs, &release.BaseConfig{
				Name:             metaRoleBaseConfigEnvName,
				Variable:         metaRoleBaseConfig.Env.Variable,
				ValueType:        metaRoleBaseConfig.Env.Type,
				ValueDescription: metaRoleBaseConfig.Env.Description,
				DefaultValue:     metaRoleBaseConfig.Env.DefaultValue,
			})
		}
		if metaRoleBaseConfig.EnvMap != nil {
			baseConfigs = append(baseConfigs, &release.BaseConfig{
				Name:             metaRoleBaseConfigEnvMapName,
				Variable:         metaRoleBaseConfig.EnvMap.Variable,
				ValueType:        metaRoleBaseConfig.EnvMap.Type,
				ValueDescription: metaRoleBaseConfig.EnvMap.Description,
				DefaultValue:     metaRoleBaseConfig.EnvMap.DefaultValue,
			})
		}
		if metaRoleBaseConfig.Image != nil {
			baseConfigs = append(baseConfigs, &release.BaseConfig{
				Name:             metaRoleBaseConfigImageName,
				Variable:         metaRoleBaseConfig.Image.Variable,
				ValueType:        metaRoleBaseConfig.Image.Type,
				ValueDescription: metaRoleBaseConfig.Image.Description,
				DefaultValue:     metaRoleBaseConfig.Image.DefaultValue,
			})
		}
	}
	return
}

func convertChartParamToBaseConfig(config *release.MetaCommonConfig) (baseConfig *release.BaseConfig, configType release.VariableType) {
	if config != nil {
		baseConfig = &release.BaseConfig{
			Name:             config.Name,
			Variable:         config.Variable,
			ValueType:        config.Type,
			ValueDescription: config.Description,
		}

		if config.DefaultValue != "" {
			baseConfig.DefaultValue = gjson.Parse(config.DefaultValue).Value()
		}

		configType = config.VariableType
	}
	return
}
