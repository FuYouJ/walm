package impl

import "WarpCloud/walm/pkg/models/release"

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
		ValueName:        baseConfig.ValueName,
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
		Name:        roleConfig.Name,
		Description: roleConfig.Description,
		Replicas:    roleConfig.Replicas,
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
		MemoryLimit: resourceConfig.MemoryLimit,
		CpuRequest: resourceConfig.CpuRequest,
		CpuLimit: resourceConfig.CpuLimit,
		GpuLimit: resourceConfig.GpuLimit,
		GpuRequest: resourceConfig.GpuRequest,
	}

	for _, storageConfig := range resourceConfig.ResourceStorageList {
		rc.ResourceStorageList = append(rc.ResourceStorageList, convertResourceStorageConfig(storageConfig))
	}
	return rc
}

func convertResourceStorageConfig(resourceStorageConfig ResourceStorageConfig) release.ResourceStorageConfig {
	return release.ResourceStorageConfig{
		Name: resourceStorageConfig.Name,
		DiskReplicas: resourceStorageConfig.DiskReplicas,
		StorageType: resourceStorageConfig.StorageType,
		AccessModes: resourceStorageConfig.AccessModes,
		Size: resourceStorageConfig.Size,
		StorageClass: resourceStorageConfig.StorageClass,
		AccessMode: resourceStorageConfig.AccessMode,
	}
}
