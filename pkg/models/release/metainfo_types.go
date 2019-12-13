package release

import (
	"encoding/json"
	"github.com/tidwall/gjson"
	"k8s.io/klog"
)

type MetaInfoCommonConfig struct {
	MapKey      string `json:"mapKey" description:"config map values.yaml key"`
	Description string `json:"description" description:"config description"`
	Type        string `json:"type" description:"config type"`
	Required    bool   `json:"required" description:"required"`
	Variable    string `json:"variable" description:"config variable (for compatible use)"`
}

func NewMetaInfoCommonConfig(mapKey, desc, configType, variable string, required bool) MetaInfoCommonConfig {
	return MetaInfoCommonConfig{
		Variable:    variable,
		Description: desc,
		Type:        configType,
		MapKey:      mapKey,
		Required:    required,
	}
}

type MetaStringConfig struct {
	MetaInfoCommonConfig
	DefaultValue string `json:"defaultValue" description:"default value of mapKey"`
}

func (config *MetaStringConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildStringConfigValue(jsonStr)
}

func (config *MetaStringConfig) BuildStringConfigValue(jsonStr string) string {
	return gjson.Get(jsonStr, config.MapKey).String()
}

func (config *MetaStringConfig) BuildConfigSet() *MetaConfigTestSet {
	return &MetaConfigTestSet{
		MapKey:   config.MapKey,
		Type:     "string",
		Required: config.Required,
	}
}

type IntConfig struct {
	MetaInfoCommonConfig
	DefaultValue int64 `json:"defaultValue" description:"default value of mapKey"`
}

type MetaIntConfig struct {
	IntConfig
}

func (config *MetaIntConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildIntConfigValue(jsonStr)
}

func (config *MetaIntConfig) BuildIntConfigValue(jsonStr string) int64 {
	return gjson.Get(jsonStr, config.MapKey).Int()
}

func (config *MetaIntConfig) BuildConfigSet() *MetaConfigTestSet {
	return &MetaConfigTestSet{
		MapKey:   config.MapKey,
		Type:     "int",
		Required: config.Required,
	}
}

type MetaEnvConfig struct {
	MetaInfoCommonConfig
	DefaultValue []MetaEnv `json:"defaultValue" description:"default value of mapKey"`
}

type MetaEnv struct {
	Name  string `json:"name" description:"env name"`
	Value string `json:"value" description:"env value"`
}

func (config *MetaEnvConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildEnvConfigValue(jsonStr)
}

func (config *MetaEnvConfig) BuildEnvConfigValue(jsonStr string) []MetaEnv {
	var metaEnv []MetaEnv
	rawMsg := gjson.Get(jsonStr, config.MapKey).Raw
	if rawMsg == "" {
		return metaEnv
	}
	err := json.Unmarshal([]byte(rawMsg), &metaEnv)
	if err != nil {
		klog.Warningf("failed to unmarshal %s : %s", rawMsg, err.Error())
	}
	return metaEnv
}

func (config *MetaEnvConfig) BuildConfigSet() *MetaConfigTestSet {
	return &MetaConfigTestSet{
		MapKey:   config.MapKey,
		Type:     "env",
		Required: config.Required,
	}
}

type MetaEnvMapConfig struct {
	MetaInfoCommonConfig
	DefaultValue map[string]string `json:"defaultValue" description:"default value of mapKey"`
}

func (config *MetaEnvMapConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildEnvConfigValue(jsonStr)
}

func (config *MetaEnvMapConfig) BuildEnvConfigValue(jsonStr string) map[string]string {
	var res map[string]string
	rawMsg := gjson.Get(jsonStr, config.MapKey).Raw
	if rawMsg == "" {
		return res
	}
	err := json.Unmarshal([]byte(rawMsg), &res)
	if err != nil {
		klog.Warningf("failed to unmarshal %s : %s", rawMsg, err.Error())
	}
	return res
}

func (config *MetaEnvMapConfig) BuildConfigSet() *MetaConfigTestSet {
	return &MetaConfigTestSet{
		MapKey:   config.MapKey,
		Type:     "envMap",
		Required: config.Required,
	}
}

type MetaBoolConfig struct {
	MetaInfoCommonConfig
	DefaultValue bool `json:"defaultValue" description:"default value of mapKey"`
}

func (config *MetaBoolConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildBoolConfigValue(jsonStr)
}

func (config *MetaBoolConfig) BuildBoolConfigValue(jsonStr string) bool {
	return gjson.Get(jsonStr, config.MapKey).Bool()
}

func (config *MetaBoolConfig) BuildConfigSet() *MetaConfigTestSet {
	return &MetaConfigTestSet{
		MapKey:   config.MapKey,
		Type:     "boolean",
		Required: config.Required,
	}
}

type FloatConfig struct {
	MetaInfoCommonConfig
	DefaultValue float64 `json:"defaultValue" description:"default value of mapKey"`
}

type MetaFloatConfig struct {
	FloatConfig
}

func (config *MetaFloatConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildFloatConfigValue(jsonStr)
}

func (config *MetaFloatConfig) BuildFloatConfigValue(jsonStr string) float64 {
	return gjson.Get(jsonStr, config.MapKey).Float()
}
