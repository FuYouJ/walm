package release

import (
	"WarpCloud/walm/pkg/models/k8s"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"k8s.io/klog"
	"reflect"
	"strconv"
	"strings"
)

// chart metainfo
type ChartMetaInfo struct {
	FriendlyName          string                     `json:"friendlyName" description:"friendlyName"`
	Categories            []string                   `json:"categories" description:"categories"`
	ChartDependenciesInfo []*ChartDependencyMetaInfo `json:"dependencies" description:"dependency metainfo"`
	ChartRoles            []*MetaRoleConfig          `json:"roles"`
	ChartParams           []*MetaCommonConfig        `json:"params"`
	Plugins               []*k8s.ReleasePlugin       `json:"plugins"`
	CustomChartParams     map[string]string          `json:"customParams"`
}

func (chartMetaInfo *ChartMetaInfo) checkDependencies() error {
	return nil
}

func (chartMetaInfo *ChartMetaInfo) CheckMetainfoValidate() ([]*MetaConfigTestSet, error) {

	var err error
	// friendlyName
	if !(len(chartMetaInfo.FriendlyName) > 0) {
		return nil, errors.Errorf("field friendlyName required")
	}

	// dependencies
	for _, dependency := range chartMetaInfo.ChartDependenciesInfo {
		if dependency != nil {
			if dependency.Name == "" || dependency.MinVersion == "" || dependency.MaxVersion == "" || dependency.AliasConfigVar == "" ||
				reflect.TypeOf(dependency.DependencyOptional).String() != "bool" {
				return nil, errors.Errorf("Name, MinVersion, MaxVersion, [aliasConfigVar,omitempty], dependencyOptional all required in field dependencies")
			}
		}
	}

	// plugins
	for pluginIndex, plugin := range chartMetaInfo.Plugins {
		if plugin != nil {
			if plugin.Name == "" || plugin.Version == "" || plugin.Args == "" {
				return nil, errors.Errorf("name, version, args all required in field plugins[%d]", pluginIndex)
			}
		}
	}

	var results []*MetaConfigTestSet
	for index, chartRole := range chartMetaInfo.ChartRoles {
		if chartRole != nil {
			configSets, err := chartRole.BuildConfigSet(index)
			if err != nil {
				return nil, err
			}
			results = append(results, configSets...)
		}
	}

	for index, chartParam := range chartMetaInfo.ChartParams {
		if chartParam != nil {
			if chartParam.Name == "" || chartParam.MapKey == "" {
				return nil, errors.Errorf("both name and mapKey required in field params[%d]", index)
			}
			switch chartParam.Type {
			case "boolean", "number", "int", "float", "string", "yaml", "json", "kvPair", "text":
			default:
				err = errors.Errorf("type <%s> not support in field params[%d]", chartParam.Type, index)
				return nil, err
			}
			results = append(results, chartParam.BuildConfigSet())
		}
	}
	return results, nil
}

func (chartMetaInfo *ChartMetaInfo) CheckParamsInValues(valuesStr string, configSets []*MetaConfigTestSet) error {
	var err error
	for _, configSet := range configSets {
		/*
			boolean --> True, False
			string --> String
			int, float --> Number
			interface, {}, [], Null --> JSON, Null
		*/
		result := gjson.Get(valuesStr, configSet.MapKey)
		if result.Exists() {
			switch configSet.Type {
			case "boolean":
				if !(result.Type.String() == "True" || result.Type.String() == "False") {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
			case "string":
				if result.Type.String() != "String" {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
				if strings.HasSuffix(configSet.MapKey, ".memory") || strings.HasSuffix(configSet.MapKey, ".memory_request") ||
					strings.HasSuffix(configSet.MapKey, ".memory_limit") {

					if !(strings.HasSuffix(result.Str, "Mi") || strings.HasSuffix(result.Str, "Gi")) {
						return errors.Errorf("%s Format error in values.yaml, eg: 4Gi, 400Mi expected", configSet.MapKey)
					}
				}
			case "int":
				_, err = strconv.Atoi(result.Raw)
				if err != nil {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
			case "float":
				if result.Type.String() != "Number" {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
			case "number":
				if result.Type.String() != "Number" {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
			case "text":
				if result.Type.String() != "String" {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
			case "kvPair":
				if result.Type.String() != "JSON" {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
			case "":
			default:
				if result.Type.String() == "Null" {
					break
				}
				if result.Type.String() != "JSON" {
					return errors.Errorf("%s Type error in values.yaml, %s expected", configSet.MapKey, configSet.Type)
				}
			}
		} else {
			if configSet.Required {
				return errors.Errorf("%s not exist in values.yaml", configSet.MapKey)
			}
		}
	}
	return nil
}

func (chartMetaInfo *ChartMetaInfo) BuildDefaultValue(jsonStr string) {
	if jsonStr != "" {
		for _, chartParam := range chartMetaInfo.ChartParams {
			if chartParam != nil {
				chartParam.BuildDefaultValue(jsonStr)
			}
		}
		for _, chartRole := range chartMetaInfo.ChartRoles {
			if chartRole != nil {
				chartRole.BuildDefaultValue(jsonStr)
			}
		}
	}
}

func (chartMetaInfo *ChartMetaInfo) BuildMetaInfoParams(configValues map[string]interface{}) (*MetaInfoParams, error) {
	if len(configValues) > 0 {
		jsonBytes, err := json.Marshal(configValues)
		if err != nil {
			klog.Errorf("failed to marshal computed values : %s", err.Error())
			return nil, err
		}
		jsonStr := string(jsonBytes)
		metaInfoValues := &MetaInfoParams{}

		for _, chartParam := range chartMetaInfo.ChartParams {
			metaInfoValues.Params = append(metaInfoValues.Params, chartParam.BuildCommonConfigValue(jsonStr))
		}

		for _, chartRole := range chartMetaInfo.ChartRoles {
			if chartRole != nil {
				metaInfoValues.Roles = append(metaInfoValues.Roles, chartRole.BuildRoleConfigValue(jsonStr))
			}
		}
		metaInfoValues.CustomChartParams = chartMetaInfo.CustomChartParams
		return metaInfoValues, nil
	}
	return nil, nil
}

type ChartDependencyMetaInfo struct {
	Name               string `json:"name,omitempty"`
	MinVersion         string `json:"minVersion"`
	MaxVersion         string `json:"maxVersion"`
	DependencyOptional bool   `json:"dependencyOptional"`
	AliasConfigVar     string `json:"aliasConfigVar,omitempty"`
	ChartName          string `json:"chartName"`
	DependencyType     string `json:"type"`
}

func (chartDependencyMetaInfo *ChartDependencyMetaInfo) AutoDependency() bool {
	if chartDependencyMetaInfo.Name == "" {
		return false
	}
	if chartDependencyMetaInfo.ChartName == "" {
		// 默认chartName = name
		return true
	}
	if chartDependencyMetaInfo.Name == chartDependencyMetaInfo.ChartName {
		return true
	}
	return false
}

type MetaRoleConfig struct {
	Name                  string                 `json:"name"`
	Description           string                 `json:"description"`
	Type                  string                 `json:"type"`
	RoleBaseConfig        *MetaRoleBaseConfig    `json:"baseConfig"`
	RoleResourceConfig    *MetaResourceConfig    `json:"resources"`
	RoleHealthCheckConfig *MetaHealthCheckConfig `json:"healthChecks"`
}

func (roleConfig *MetaRoleConfig) BuildDefaultValue(jsonStr string) {
	if roleConfig.RoleBaseConfig != nil {
		roleConfig.RoleBaseConfig.BuildDefaultValue(jsonStr)
	}
	if roleConfig.RoleResourceConfig != nil {
		roleConfig.RoleResourceConfig.BuildDefaultValue(jsonStr)
	}
}

func (roleConfig *MetaRoleConfig) BuildRoleConfigValue(jsonStr string) *MetaRoleConfigValue {
	roleConfigValue := &MetaRoleConfigValue{Name: roleConfig.Name}
	if roleConfig.RoleBaseConfig != nil {
		roleConfigValue.RoleBaseConfigValue = roleConfig.RoleBaseConfig.BuildRoleBaseConfigValue(jsonStr)
	}
	if roleConfig.RoleResourceConfig != nil {
		roleConfigValue.RoleResourceConfigValue = roleConfig.RoleResourceConfig.BuildResourceConfigValue(jsonStr)
	}
	return roleConfigValue
}

func (roleConfig *MetaRoleConfig) BuildConfigSet(index int) ([]*MetaConfigTestSet, error) {
	var configSets []*MetaConfigTestSet
	if roleConfig.RoleBaseConfig != nil {
		baseConfigSets, err := roleConfig.RoleBaseConfig.BuildConfigSet(index)
		if err != nil {
			return nil, err
		}
		configSets = append(configSets, baseConfigSets...)
	}
	if roleConfig.RoleResourceConfig != nil {
		resourceConfigSets, err := roleConfig.RoleResourceConfig.BuildConfigSet(index)
		if err != nil {
			return nil, err
		}
		configSets = append(configSets, resourceConfigSets...)

	}
	return configSets, nil
}

func (roleBaseConfig *MetaRoleBaseConfig) BuildConfigSet(index int) ([]*MetaConfigTestSet, error) {
	var configSets []*MetaConfigTestSet
	if roleBaseConfig.Image != nil {
		if roleBaseConfig.Image.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].baseConfig.image", index)
		}
		configSets = append(configSets, roleBaseConfig.Image.BuildConfigSet())
	}
	if roleBaseConfig.Replicas != nil {
		if roleBaseConfig.Replicas.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].baseConfig.replicas", index)
		}
		configSets = append(configSets, roleBaseConfig.Replicas.BuildConfigSet())
	}
	if roleBaseConfig.Env != nil {
		if roleBaseConfig.Env.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].baseConfig.env", index)
		}
		configSets = append(configSets, roleBaseConfig.Env.BuildConfigSet())
	}
	if roleBaseConfig.EnvMap != nil {
		if roleBaseConfig.EnvMap.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].baseConfig.envMap", index)
		}
		configSets = append(configSets, roleBaseConfig.EnvMap.BuildConfigSet())
	}
	if roleBaseConfig.UseHostNetwork != nil {
		if roleBaseConfig.UseHostNetwork.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].baseConfig.useHostNetwork:", index)
		}
		configSets = append(configSets, roleBaseConfig.UseHostNetwork.BuildConfigSet())
	}
	if roleBaseConfig.Priority != nil {
		if roleBaseConfig.Priority.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].baseConfig.priority", index)
		}
		configSets = append(configSets, roleBaseConfig.Priority.BuildConfigSet())
	}
	for otherIndex, config := range roleBaseConfig.Others {
		if config != nil {
			if config.Name == "" || config.MapKey == "" {
				return nil, errors.Errorf("both name and mapKey required in field roles[%d].baseConfig.others[%d]", index, otherIndex)
			}
			switch config.Type {
			case "boolean", "int", "float", "string", "yaml", "json", "kvPair", "text":
			default:
				return nil, errors.Errorf("type <%s> not support in field roles[%d].baseConfig.others[%d]", config.Type, index, otherIndex)
			}
			configSets = append(configSets, config.BuildConfigSet())
		}
	}
	return configSets, nil
}

func (resConfig *MetaResourceConfig) BuildConfigSet(index int) ([]*MetaConfigTestSet, error) {
	var configSets []*MetaConfigTestSet
	if resConfig.LimitsCpu != nil {
		if resConfig.LimitsCpu.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].resources.limitsCpu", index)
		}
		configSets = append(configSets, resConfig.LimitsCpu.BuildConfigSet())
	}
	if resConfig.LimitsGpu != nil {
		if resConfig.LimitsGpu.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].resources.limitsGpu", index)
		}
		configSets = append(configSets, resConfig.LimitsGpu.BuildConfigSet())
	}
	if resConfig.LimitsMemory != nil {
		if resConfig.LimitsMemory.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].resources.limitsMemory", index)
		}
		configSets = append(configSets, resConfig.LimitsMemory.BuildConfigSet())
	}
	if resConfig.RequestsCpu != nil {
		if resConfig.RequestsCpu.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].resources.requestsCpu", index)
		}
		configSets = append(configSets, resConfig.RequestsCpu.BuildConfigSet())
	}
	if resConfig.RequestsGpu != nil {
		if resConfig.RequestsGpu.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].resources.requestsGpu", index)
		}
		configSets = append(configSets, resConfig.RequestsGpu.BuildConfigSet())
	}
	if resConfig.RequestsMemory != nil {
		if resConfig.RequestsMemory.MapKey == "" {
			return nil, errors.Errorf("mapKey required in field roles[%d].resources.requestsMemory", index)
		}
		configSets = append(configSets, resConfig.RequestsMemory.BuildConfigSet())
	}
	if resConfig.StorageResources != nil {
		for storageIndex, storageResource := range resConfig.StorageResources {
			if storageResource != nil {
				if storageResource.MapKey == "" || storageResource.Name == "" {
					return nil, errors.Errorf("both name and mapKey required in field roles[%d].resources.storageResources[%d]", index, storageIndex)
				}
				configSets = append(configSets, storageResource.BuildConfigSet())
			}
		}
	}
	return configSets, nil
}

type MetaRoleBaseConfig struct {
	Image          *MetaStringConfig   `json:"image" description:"role image"`
	Priority       *MetaIntConfig      `json:"priority" description:"role priority"`
	Replicas       *MetaIntConfig      `json:"replicas" description:"role replicas"`
	Env            *MetaEnvConfig      `json:"env" description:"role env list"`
	EnvMap         *MetaEnvMapConfig   `json:"envMap" description:"role env map"`
	UseHostNetwork *MetaBoolConfig     `json:"useHostNetwork" description:"whether role use host network"`
	Others         []*MetaCommonConfig `json:"others" description:"role other configs"`
}

func (roleBaseConfig *MetaRoleBaseConfig) BuildDefaultValue(jsonStr string) {
	if roleBaseConfig.Image != nil {
		roleBaseConfig.Image.BuildDefaultValue(jsonStr)
	}
	if roleBaseConfig.Replicas != nil {
		roleBaseConfig.Replicas.BuildDefaultValue(jsonStr)
	}
	if roleBaseConfig.Env != nil {
		roleBaseConfig.Env.BuildDefaultValue(jsonStr)
	}
	if roleBaseConfig.EnvMap != nil {
		roleBaseConfig.EnvMap.BuildDefaultValue(jsonStr)
	}
	if roleBaseConfig.UseHostNetwork != nil {
		roleBaseConfig.UseHostNetwork.BuildDefaultValue(jsonStr)
	}
	if roleBaseConfig.Priority != nil {
		roleBaseConfig.Priority.BuildDefaultValue(jsonStr)
	}
	for _, config := range roleBaseConfig.Others {
		if config != nil {
			config.BuildDefaultValue(jsonStr)
		}
	}
}

func (roleBaseConfig *MetaRoleBaseConfig) BuildRoleBaseConfigValue(jsonStr string) *MetaRoleBaseConfigValue {
	roleBaseConfigValue := &MetaRoleBaseConfigValue{}
	if roleBaseConfig.Image != nil {
		image := roleBaseConfig.Image.BuildStringConfigValue(jsonStr)
		roleBaseConfigValue.Image = &image
	}
	if roleBaseConfig.Replicas != nil {
		replicas := roleBaseConfig.Replicas.BuildIntConfigValue(jsonStr)
		roleBaseConfigValue.Replicas = &replicas
	}
	if roleBaseConfig.Env != nil {
		roleBaseConfigValue.Env = roleBaseConfig.Env.BuildEnvConfigValue(jsonStr)
	}
	if roleBaseConfig.EnvMap != nil {
		roleBaseConfigValue.EnvMap = roleBaseConfig.EnvMap.BuildEnvConfigValue(jsonStr)
	}
	if roleBaseConfig.UseHostNetwork != nil {
		useHostNetwork := roleBaseConfig.UseHostNetwork.BuildBoolConfigValue(jsonStr)
		roleBaseConfigValue.UseHostNetwork = &useHostNetwork
	}
	if roleBaseConfig.Priority != nil {
		priority := roleBaseConfig.Priority.BuildIntConfigValue(jsonStr)
		roleBaseConfigValue.Priority = &priority
	}
	for _, config := range roleBaseConfig.Others {
		if config != nil {
			roleBaseConfigValue.Others = append(roleBaseConfigValue.Others, config.BuildCommonConfigValue(jsonStr))
		}
	}
	return roleBaseConfigValue
}

type MetaResourceConfig struct {
	LimitsMemory     *MetaResourceMemoryConfig    `json:"limitsMemory" description:"resource memory limit"`
	LimitsCpu        *MetaResourceCpuConfig       `json:"limitsCpu" description:"resource cpu limit"`
	LimitsGpu        *MetaResourceCpuConfig       `json:"limitsGpu" description:"resource gpu limit"`
	RequestsMemory   *MetaResourceMemoryConfig    `json:"requestsMemory" description:"resource memory request"`
	RequestsCpu      *MetaResourceCpuConfig       `json:"requestsCpu" description:"resource cpu request"`
	RequestsGpu      *MetaResourceCpuConfig       `json:"requestsGpu" description:"resource gpu request"`
	StorageResources []*MetaResourceStorageConfig `json:"storageResources" description:"resource storage request"`
}

func (config *MetaResourceConfig) BuildDefaultValue(jsonStr string) {
	if config.LimitsMemory != nil {
		config.LimitsMemory.BuildDefaultValue(jsonStr)
	}
	if config.LimitsGpu != nil {
		config.LimitsGpu.BuildDefaultValue(jsonStr)
	}
	if config.LimitsCpu != nil {
		config.LimitsCpu.BuildDefaultValue(jsonStr)
	}
	if config.RequestsMemory != nil {
		config.RequestsMemory.BuildDefaultValue(jsonStr)
	}
	if config.RequestsGpu != nil {
		config.RequestsGpu.BuildDefaultValue(jsonStr)
	}
	if config.RequestsCpu != nil {
		config.RequestsCpu.BuildDefaultValue(jsonStr)
	}

	for _, storageConfig := range config.StorageResources {
		if storageConfig != nil {
			storageConfig.BuildDefaultValue(jsonStr)
		}
	}
}

func (config *MetaResourceConfig) BuildResourceConfigValue(jsonStr string) *MetaResourceConfigValue {
	resourceConfigValue := &MetaResourceConfigValue{}
	if config.LimitsMemory != nil {
		limitsMemory := config.LimitsMemory.BuildMemoryConfigValue(jsonStr)
		resourceConfigValue.LimitsMemory = &limitsMemory
	}
	if config.LimitsGpu != nil {
		limitsGpu := config.LimitsGpu.BuildCpuConfigValue(jsonStr)
		resourceConfigValue.LimitsGpu = &limitsGpu
	}
	if config.LimitsCpu != nil {
		limitsCpu := config.LimitsCpu.BuildCpuConfigValue(jsonStr)
		resourceConfigValue.LimitsCpu = &limitsCpu
	}
	if config.RequestsMemory != nil {
		requestsMemory := config.RequestsMemory.BuildMemoryConfigValue(jsonStr)
		resourceConfigValue.RequestsMemory = &requestsMemory
	}
	if config.RequestsGpu != nil {
		requestsGpu := config.RequestsGpu.BuildCpuConfigValue(jsonStr)
		resourceConfigValue.RequestsGpu = &requestsGpu
	}
	if config.RequestsCpu != nil {
		requestsCpu := config.RequestsCpu.BuildCpuConfigValue(jsonStr)
		resourceConfigValue.RequestsCpu = &requestsCpu
	}

	for _, storageConfig := range config.StorageResources {
		if storageConfig != nil {
			resourceConfigValue.StorageResources = append(resourceConfigValue.StorageResources, storageConfig.BuildStorageConfigValue(jsonStr))
		}
	}
	return resourceConfigValue
}

type VariableType string

const (
	AdvanceConfig         VariableType = "advanceConfig"
	TranswarpBundleConfig VariableType = "transwarpBundleConfig"
)

type MetaCommonConfig struct {
	MetaInfoCommonConfig
	Name         string       `json:"name" description:"config name"`
	DefaultValue string       `json:"defaultValue" description:"default value of mapKey"`
	VariableType VariableType `json:"variableType" description:"config variable type: advanceConfig, transwarpBundleConfig (for compatible use)"`
}

func (config *MetaCommonConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildCommonConfigValue(jsonStr).Value
}

func (config *MetaCommonConfig) BuildCommonConfigValue(jsonStr string) *MetaCommonConfigValue {
	return &MetaCommonConfigValue{
		Name:  config.Name,
		Type:  config.Type,
		Value: gjson.Get(jsonStr, config.MapKey).Raw,
	}
}

func (config *MetaCommonConfig) BuildConfigSet() *MetaConfigTestSet {
	return &MetaConfigTestSet{
		MapKey:   config.MapKey,
		Type:     config.Type,
		Required: config.Required,
	}
}

type MetaHealthProbeConfig struct {
	Defined bool `json:"defined" description:"health check is defined"`
	Enable  bool `json:"enable" description:"enable health check"`
}

type MetaHealthCheckConfig struct {
	ReadinessProbe *MetaHealthProbeConfig `json:"readinessProbe"`
	LivenessProbe  *MetaHealthProbeConfig `json:"livenessProbe"`
}
