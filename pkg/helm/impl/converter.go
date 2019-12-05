package impl

import (
	"WarpCloud/walm/pkg/models/release"
	"github.com/tidwall/gjson"
	"WarpCloud/walm/pkg/k8s/utils"
	"encoding/json"
	"k8s.io/klog"
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

func convertPrettyParamsToMetainfoParams(metainfo *release.ChartMetaInfo, prettyParams *release.PrettyChartParams) (*release.MetaInfoParams, error) {
	klog.Info("converting pretty params to meta info params")
	if metainfo == nil || prettyParams == nil {
		return nil, nil
	}

	metaInfoParams := &release.MetaInfoParams{}

	for _, roleConfig := range prettyParams.CommonConfig.Roles {
		if roleConfig != nil {
			metaRoleConfig := getMetaRoleConfig(metainfo, roleConfig.Name)
			metaRoleConfigValue, err := computeMetaRoleConfigValue(metaRoleConfig, roleConfig)
			if err != nil {
				klog.Errorf("failed to compute meta role config value : %s", err.Error())
				return nil, err
			}
			metaInfoParams.Roles = append(metaInfoParams.Roles, metaRoleConfigValue)
		}
	}

	baseConfigs := []*release.BaseConfig{}
	baseConfigs = append(baseConfigs, prettyParams.AdvanceConfig...)
	baseConfigs = append(baseConfigs, prettyParams.TranswarpBaseConfig...)

	for _, baseConfig := range baseConfigs {
		if baseConfig != nil {
			metaCommonConfigValue, err := convertMetaCommonConfigValue(metainfo.ChartParams, baseConfig)
			if err != nil {
				klog.Errorf("failed to convert meta common config value : %s", err.Error())
				return nil, err
			}
			metaInfoParams.Params = append(metaInfoParams.Params, metaCommonConfigValue)
		}
	}

	return metaInfoParams, nil
}

func convertMetaCommonConfigValue(commonConfigs []*release.MetaCommonConfig, baseConfig *release.BaseConfig) (*release.MetaCommonConfigValue, error) {
	commonConfig := getCommonConfig(commonConfigs, baseConfig)
	rawJsonStr := []byte{}
	var err error
	if baseConfig.DefaultValue != nil {
		rawJsonStr, err = json.Marshal(baseConfig.DefaultValue)
		if err != nil {
			klog.Errorf("failed to marshal : %s", err.Error())
			return nil, err
		}
	}

	return &release.MetaCommonConfigValue{
		Name:  commonConfig.Name,
		Type:  commonConfig.Type,
		Value: string(rawJsonStr),
	}, nil
}

func getCommonConfig(metaCommonConfigs []*release.MetaCommonConfig, baseConfig *release.BaseConfig) *release.MetaCommonConfig {
	for _, commonConfig := range metaCommonConfigs {
		if commonConfig != nil {
			if baseConfig.Name != "" {
				if baseConfig.Name == commonConfig.Name {
					return commonConfig
				}
			} else if baseConfig.Variable != "" {
				if baseConfig.Variable == baseConfig.Variable {
					return commonConfig
				}
			}
		}
	}
	return nil
}

func computeMetaRoleConfigValue(metaRoleConfig *release.MetaRoleConfig, roleConfig *release.RoleConfig) (metaRoleConfigValue *release.MetaRoleConfigValue, err error) {
	if metaRoleConfig != nil {
		metaRoleConfigValue = &release.MetaRoleConfigValue{
			Name: roleConfig.Name,
		}
		for _, baseConfig := range roleConfig.RoleBaseConfig {
			if baseConfig != nil {
				err = fillMetaRoleBaseConfigValue(metaRoleConfigValue, metaRoleConfig, baseConfig)
				if err != nil {
					klog.Errorf("failed to fill meta role config value : %s", err.Error())
					return
				}
			}
		}
		if roleConfig.RoleResourceConfig != nil {
			metaRoleConfigValue.RoleResourceConfigValue = &release.MetaResourceConfigValue{}
			err = fillMetaResourceConfigValue(metaRoleConfigValue, metaRoleConfig, roleConfig.RoleResourceConfig)
			if err != nil {
				klog.Errorf("failed to fill meta resource config value : %s", err.Error())
				return
			}
		}
	}
	return
}

func fillMetaResourceConfigValue(metaRoleConfigValue *release.MetaRoleConfigValue, metaRoleConfig *release.MetaRoleConfig, resourceConfig *release.ResourceConfig) error {
	if resourceConfig.MemoryRequest != 0 {
		value := int64(resourceConfig.MemoryRequest)
		metaRoleConfigValue.RoleResourceConfigValue.RequestsMemory = &value
	}
	if resourceConfig.MemoryLimit != 0 {
		value := int64(resourceConfig.MemoryLimit)
		metaRoleConfigValue.RoleResourceConfigValue.LimitsMemory = &value
	}
	if resourceConfig.CpuRequest != 0 {
		metaRoleConfigValue.RoleResourceConfigValue.RequestsCpu = &resourceConfig.CpuRequest
	}
	if resourceConfig.CpuLimit != 0 {
		metaRoleConfigValue.RoleResourceConfigValue.LimitsCpu = &resourceConfig.CpuLimit
	}
	if resourceConfig.GpuRequest != 0 {
		value := float64(resourceConfig.GpuRequest)
		metaRoleConfigValue.RoleResourceConfigValue.RequestsGpu = &value
	}
	if resourceConfig.GpuLimit != 0 {
		value := float64(resourceConfig.GpuLimit)
		metaRoleConfigValue.RoleResourceConfigValue.LimitsGpu = &value
	}

	for _, resourceStorage := range resourceConfig.ResourceStorageList {
		resourceStorageConfigValue := convertResourceStorageConfigValue(resourceStorage)
		metaRoleConfigValue.RoleResourceConfigValue.StorageResources =
			append(metaRoleConfigValue.RoleResourceConfigValue.StorageResources, resourceStorageConfigValue)
	}

	return nil
}

func convertResourceStorageConfigValue(resourceStorageConfig release.ResourceStorageConfig) *release.MetaResourceStorageConfigValue {
	return &release.MetaResourceStorageConfigValue{
		Name: resourceStorageConfig.Name,
		Value: &release.MetaResourceStorage{
			ResourceStorage: release.ResourceStorage{
				DiskReplicas: resourceStorageConfig.DiskReplicas,
				AccessModes:  resourceStorageConfig.AccessModes,
				StorageClass: resourceStorageConfig.StorageClass,
			},
			Size: utils.ParseK8sResourceStorage(resourceStorageConfig.Size),
		},
	}
}

func fillMetaRoleBaseConfigValue(metaRoleConfigValue *release.MetaRoleConfigValue, metaRoleConfig *release.MetaRoleConfig, baseConfig *release.BaseConfig) (err error) {
	if metaRoleConfigValue.RoleBaseConfigValue == nil {
		metaRoleConfigValue.RoleBaseConfigValue = &release.MetaRoleBaseConfigValue{}
	}
	if baseConfig.Name != "" {
		switch baseConfig.Name {
		case metaRoleBaseConfigReplicasName:
			metaRoleConfigValue.RoleBaseConfigValue.Replicas, err = computeInt64(baseConfig.DefaultValue)
		case metaRoleBaseConfigUseHostNetworkName:
			metaRoleConfigValue.RoleBaseConfigValue.UseHostNetwork, err = computeBool(baseConfig.DefaultValue)
		case metaRoleBaseConfigPriorityName:
			metaRoleConfigValue.RoleBaseConfigValue.Priority, err = computeInt64(baseConfig.DefaultValue)
		case metaRoleBaseConfigEnvName:
			metaRoleConfigValue.RoleBaseConfigValue.Env, err = computeMetaEnv(baseConfig.DefaultValue)
		case metaRoleBaseConfigEnvMapName:
			metaRoleConfigValue.RoleBaseConfigValue.EnvMap, err = computeMap(baseConfig.DefaultValue)
		case metaRoleBaseConfigImageName:
			metaRoleConfigValue.RoleBaseConfigValue.Image, err = computeString(baseConfig.DefaultValue)
		}
	} else if baseConfig.Variable != "" {
		replicasVariable, useHostNetworkVariable, priorityVariable, envListVariable, envMapVariable, imageVariable :=
			buildMetaRoleBaseConfigVariables(metaRoleConfig)
		switch baseConfig.Variable {
		case replicasVariable:
			metaRoleConfigValue.RoleBaseConfigValue.Replicas, err = computeInt64(baseConfig.DefaultValue)
		case useHostNetworkVariable:
			metaRoleConfigValue.RoleBaseConfigValue.UseHostNetwork, err = computeBool(baseConfig.DefaultValue)
		case priorityVariable:
			metaRoleConfigValue.RoleBaseConfigValue.Priority, err = computeInt64(baseConfig.DefaultValue)
		case envListVariable:
			metaRoleConfigValue.RoleBaseConfigValue.Env, err = computeMetaEnv(baseConfig.DefaultValue)
		case envMapVariable:
			metaRoleConfigValue.RoleBaseConfigValue.EnvMap, err = computeMap(baseConfig.DefaultValue)
		case imageVariable:
			metaRoleConfigValue.RoleBaseConfigValue.Image, err = computeString(baseConfig.DefaultValue)
		}
	}
	if err != nil {
		klog.Errorf("failed to compute role base config value : %s", err.Error())
		return
	}
	return
}

func buildMetaRoleBaseConfigVariables(metaRoleConfig *release.MetaRoleConfig) (replicasVariable,
useHostNetworkVariable, priorityVariable, envListVariable, envMapVariable, imageVariable string) {
	if metaRoleConfig != nil && metaRoleConfig.RoleBaseConfig != nil {
		if metaRoleConfig.RoleBaseConfig.Replicas != nil {
			replicasVariable = metaRoleConfig.RoleBaseConfig.Replicas.Variable
		}
		if metaRoleConfig.RoleBaseConfig.UseHostNetwork != nil {
			useHostNetworkVariable = metaRoleConfig.RoleBaseConfig.UseHostNetwork.Variable
		}
		if metaRoleConfig.RoleBaseConfig.Priority != nil {
			priorityVariable = metaRoleConfig.RoleBaseConfig.Priority.Variable
		}
		if metaRoleConfig.RoleBaseConfig.Env != nil {
			envListVariable = metaRoleConfig.RoleBaseConfig.Env.Variable
		}
		if metaRoleConfig.RoleBaseConfig.EnvMap != nil {
			envMapVariable = metaRoleConfig.RoleBaseConfig.EnvMap.Variable
		}
		if metaRoleConfig.RoleBaseConfig.Image != nil {
			imageVariable = metaRoleConfig.RoleBaseConfig.Image.Variable
		}
	}
	return
}

func computeInt64(value interface{}) (*int64, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		klog.Errorf("failed to marshal : %s", err.Error())
		return nil, err
	}
	var res int64
	err = json.Unmarshal(valueBytes, &res)
	if err != nil {
		klog.Errorf("failed to unmarshal : %s", err.Error())
		return nil, err
	}
	return &res, nil
}

func computeBool(value interface{}) (*bool, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		klog.Errorf("failed to marshal : %s", err.Error())
		return nil, err
	}
	var res bool
	err = json.Unmarshal(valueBytes, &res)
	if err != nil {
		klog.Errorf("failed to unmarshal : %s", err.Error())
		return nil, err
	}
	return &res, nil
}

func computeMetaEnv(value interface{}) ([]release.MetaEnv, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		klog.Errorf("failed to marshal : %s", err.Error())
		return nil, err
	}
	var res []release.MetaEnv
	err = json.Unmarshal(valueBytes, &res)
	if err != nil {
		klog.Errorf("failed to unmarshal : %s", err.Error())
		return nil, err
	}
	return res, nil
}

func computeMap(value interface{}) (map[string]string, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		klog.Errorf("failed to marshal : %s", err.Error())
		return nil, err
	}
	var res map[string]string
	err = json.Unmarshal(valueBytes, &res)
	if err != nil {
		klog.Errorf("failed to unmarshal : %s", err.Error())
		return nil, err
	}
	return res, nil
}

func computeString(value interface{}) (*string, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		klog.Errorf("failed to marshal : %s", err.Error())
		return nil, err
	}
	var res string
	err = json.Unmarshal(valueBytes, &res)
	if err != nil {
		klog.Errorf("failed to unmarshal : %s", err.Error())
		return nil, err
	}
	return &res, nil
}

func getMetaRoleConfig(chartMetaInfo *release.ChartMetaInfo, roleName string) *release.MetaRoleConfig {
	for _, metaRoleConfig := range chartMetaInfo.ChartRoles {
		if metaRoleConfig != nil && metaRoleConfig.Name == roleName {
			return metaRoleConfig
		}
	}
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
			Name:               metaRoleConfig.Name,
			Description:        metaRoleConfig.Description,
			RoleBaseConfig:     convertMetaRoleBaseConfigToBaseConfigs(metaRoleConfig.RoleBaseConfig),
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

func convertMetaInfoParamsToPrettyParams(metaInfo *release.ChartMetaInfo, metaInfoParams *release.MetaInfoParams) *release.PrettyChartParams {
	if metaInfo == nil || metaInfoParams == nil {
		return nil
	}

	prettyParams := &release.PrettyChartParams{}

	for _, metaRoleConfigValue := range metaInfoParams.Roles {
		if metaRoleConfigValue != nil {
			metaRoleConfig := getMetaRoleConfig(metaInfo, metaRoleConfigValue.Name)
			roleConfig := computeRoleConfig(metaRoleConfig, metaRoleConfigValue)
			prettyParams.CommonConfig.Roles = append(prettyParams.CommonConfig.Roles, roleConfig)
		}
	}

	for _, metaCommonConfigValue := range metaInfoParams.Params {
		if metaCommonConfigValue != nil {
			metaCommonConfig := getCommonConfigByName(metaInfo.ChartParams, metaCommonConfigValue.Name)
			if metaCommonConfig != nil {
				config := &release.BaseConfig{
					Name:             metaCommonConfig.Name,
					Variable:         metaCommonConfig.Variable,
					ValueType:        metaCommonConfig.Type,
					ValueDescription: metaCommonConfig.Description,
				}
				if metaCommonConfigValue.Value != "" {
					config.DefaultValue = gjson.Parse(metaCommonConfigValue.Value).Value()
				}
				switch metaCommonConfig.VariableType {
				case release.AdvanceConfig:
					prettyParams.AdvanceConfig = append(prettyParams.AdvanceConfig, config)
				case release.TranswarpBundleConfig:
					prettyParams.TranswarpBaseConfig = append(prettyParams.TranswarpBaseConfig, config)
				default:
					prettyParams.AdvanceConfig = append(prettyParams.AdvanceConfig, config)
				}
			}
		}
	}
	return prettyParams
}
func getCommonConfigByName(metaCommonConfigs []*release.MetaCommonConfig, name string) *release.MetaCommonConfig {
	for _, conf := range metaCommonConfigs {
		if conf != nil && conf.Name == name {
			return conf
		}
	}
	return nil
}

func computeRoleConfig(metaRoleConfig *release.MetaRoleConfig, metaRoleConfigValue *release.MetaRoleConfigValue) (roleConfig *release.RoleConfig) {
	if metaRoleConfig != nil {
		roleConfig = &release.RoleConfig{
			Name:        metaRoleConfig.Name,
			Description: metaRoleConfig.Description,
		}
		if metaRoleConfigValue.RoleBaseConfigValue != nil {
			if metaRoleConfigValue.RoleBaseConfigValue.Replicas != nil && metaRoleConfig.RoleBaseConfig.Replicas != nil {
				roleConfig.RoleBaseConfig = append(roleConfig.RoleBaseConfig, newBaseConfig(metaRoleBaseConfigReplicasName,
					metaRoleConfig.RoleBaseConfig.Replicas.MetaInfoCommonConfig,
					*metaRoleConfigValue.RoleBaseConfigValue.Replicas))
			}
			if metaRoleConfigValue.RoleBaseConfigValue.UseHostNetwork != nil && metaRoleConfig.RoleBaseConfig.UseHostNetwork != nil {
				roleConfig.RoleBaseConfig = append(roleConfig.RoleBaseConfig, newBaseConfig(metaRoleBaseConfigUseHostNetworkName,
					metaRoleConfig.RoleBaseConfig.UseHostNetwork.MetaInfoCommonConfig,
					*metaRoleConfigValue.RoleBaseConfigValue.UseHostNetwork))
			}
			if metaRoleConfigValue.RoleBaseConfigValue.Priority != nil && metaRoleConfig.RoleBaseConfig.Priority != nil {
				roleConfig.RoleBaseConfig = append(roleConfig.RoleBaseConfig, newBaseConfig(metaRoleBaseConfigPriorityName,
					metaRoleConfig.RoleBaseConfig.Priority.MetaInfoCommonConfig,
					*metaRoleConfigValue.RoleBaseConfigValue.Priority))
			}
			if metaRoleConfigValue.RoleBaseConfigValue.Env != nil && metaRoleConfig.RoleBaseConfig.Env != nil {
				roleConfig.RoleBaseConfig = append(roleConfig.RoleBaseConfig, newBaseConfig(metaRoleBaseConfigEnvName,
					metaRoleConfig.RoleBaseConfig.Env.MetaInfoCommonConfig,
					metaRoleConfigValue.RoleBaseConfigValue.Env))
			}
			if metaRoleConfigValue.RoleBaseConfigValue.EnvMap != nil && metaRoleConfig.RoleBaseConfig.EnvMap != nil {
				roleConfig.RoleBaseConfig = append(roleConfig.RoleBaseConfig, newBaseConfig(metaRoleBaseConfigEnvMapName,
					metaRoleConfig.RoleBaseConfig.EnvMap.MetaInfoCommonConfig,
					metaRoleConfigValue.RoleBaseConfigValue.EnvMap))
			}
			if metaRoleConfigValue.RoleBaseConfigValue.Image != nil && metaRoleConfig.RoleBaseConfig.Image != nil {
				roleConfig.RoleBaseConfig = append(roleConfig.RoleBaseConfig, newBaseConfig(metaRoleBaseConfigImageName,
					metaRoleConfig.RoleBaseConfig.Image.MetaInfoCommonConfig,
					*metaRoleConfigValue.RoleBaseConfigValue.Image))
			}
		}
		if metaRoleConfigValue.RoleResourceConfigValue != nil {
			roleConfig.RoleResourceConfig = &release.ResourceConfig{}
			if metaRoleConfigValue.RoleResourceConfigValue.RequestsMemory != nil {
				roleConfig.RoleResourceConfig.MemoryRequest = float64(*metaRoleConfigValue.RoleResourceConfigValue.RequestsMemory)
			}
			if metaRoleConfigValue.RoleResourceConfigValue.LimitsMemory != nil {
				roleConfig.RoleResourceConfig.MemoryLimit = float64(*metaRoleConfigValue.RoleResourceConfigValue.LimitsMemory)
			}
			if metaRoleConfigValue.RoleResourceConfigValue.RequestsGpu != nil {
				roleConfig.RoleResourceConfig.GpuRequest = int(*metaRoleConfigValue.RoleResourceConfigValue.RequestsGpu)
			}
			if metaRoleConfigValue.RoleResourceConfigValue.LimitsGpu != nil {
				roleConfig.RoleResourceConfig.GpuLimit = int(*metaRoleConfigValue.RoleResourceConfigValue.LimitsGpu)
			}
			if metaRoleConfigValue.RoleResourceConfigValue.RequestsCpu != nil {
				roleConfig.RoleResourceConfig.CpuRequest = float64(*metaRoleConfigValue.RoleResourceConfigValue.RequestsCpu)
			}
			if metaRoleConfigValue.RoleResourceConfigValue.LimitsCpu != nil {
				roleConfig.RoleResourceConfig.CpuLimit = float64(*metaRoleConfigValue.RoleResourceConfigValue.LimitsCpu)
			}
			for _, metaResourceConfigValue := range metaRoleConfigValue.RoleResourceConfigValue.StorageResources {
				if metaResourceConfigValue != nil && metaResourceConfigValue.Value != nil {
					resourceStorageConfig := release.ResourceStorageConfig{
						Name:         metaResourceConfigValue.Name,
						Size:         release.ConvertResourceBinaryIntByUnit(&metaResourceConfigValue.Value.Size, utils.K8sResourceStorageUnit),
						StorageClass: metaResourceConfigValue.Value.StorageClass,
						AccessModes:  metaResourceConfigValue.Value.AccessModes,
						DiskReplicas: metaResourceConfigValue.Value.DiskReplicas,
					}
					roleConfig.RoleResourceConfig.ResourceStorageList =
						append(roleConfig.RoleResourceConfig.ResourceStorageList, resourceStorageConfig)
				}
			}
		}
	}
	return
}

func newBaseConfig(name string, metaInfoCommonConfig release.MetaInfoCommonConfig, value interface{}) *release.BaseConfig {
	return &release.BaseConfig{
		Name:             name,
		Variable:         metaInfoCommonConfig.Variable,
		ValueType:        metaInfoCommonConfig.Type,
		ValueDescription: metaInfoCommonConfig.Description,
		DefaultValue:     value,
	}
}
