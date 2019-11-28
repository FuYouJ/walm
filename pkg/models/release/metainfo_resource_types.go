package release

import (
	"WarpCloud/walm/pkg/k8s/utils"
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"k8s.io/klog"
	"strconv"
)

type MetaResourceMemoryConfig struct {
	IntConfig
}

func (config *MetaResourceMemoryConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildMemoryConfigValue(jsonStr)
}

func (config *MetaResourceMemoryConfig) BuildMemoryConfigValue(jsonStr string) int64 {
	strValue := getResourceStr(jsonStr, config.MapKey)
	if strValue == "" {
		return 0
	}
	return utils.ParseK8sResourceMemory(strValue)
}

func getResourceStr(jsonStr, mapKey string) string {
	if jsonStr == "" || mapKey == "" {
		return ""
	}
	return gjson.Get(jsonStr, mapKey).String()
}

type MetaResourceCpuConfig struct {
	FloatConfig
}

func (config *MetaResourceCpuConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildCpuConfigValue(jsonStr)
}

func (config *MetaResourceCpuConfig) BuildCpuConfigValue(jsonStr string) float64 {
	strValue := getResourceStr(jsonStr, config.MapKey)
	if strValue == "" {
		return 0
	}
	return utils.ParseK8sResourceCpu(strValue)
}

type ResourceStorage struct {
	AccessModes  []string `json:"accessModes, omitempty" description:"storage access modes"`
	StorageClass string   `json:"storageClass" description:"storage class"`
	DiskReplicas int      `json:"diskReplicas"`
}

type MetaResourceStorage struct {
	ResourceStorage
	Size int64 `json:"size" description:"storage size"`
}

type MetaResourceStorageWithStringSize struct {
	ResourceStorage
	Size string `json:"size" description:"storage size"`
}

type MetaResourceStorageConfig struct {
	Name               string               `json:"name" description:"config name"`
	MapKey             string               `json:"mapKey" description:"config map values.yaml key"`
	DefaultValue       *MetaResourceStorage `json:"defaultValue" description:"default value of mapKey"`
	Description        string               `json:"description" description:"config description"`
	Type               string               `json:"type" description:"config type"`
	Required           bool                 `json:"required" description:"required"`
	AccessModeMapKey   string               `json:"accessModeMapKey"`
	StorageClassMapKey string               `json:"storageClassMapKey"`
	SizeMapKey         string               `json:"sizeMapKey"`
}

type MetaConfigTestSet struct {
	MapKey   string `json:"mapKey" description:"config map values.yaml key"`
	Type     string `json:"type" description:"config type"`
	Required bool   `json:"required" description:"required"`
}

func (config *MetaResourceStorageConfig) BuildDefaultValue(jsonStr string) {
	config.DefaultValue = config.BuildStorageConfigValue(jsonStr).Value
}

func (config *MetaResourceStorageConfig) BuildStorageConfigValue(jsonStr string) *MetaResourceStorageConfigValue {
	resourceStorageConfigValue := &MetaResourceStorageConfigValue{
		Name: config.Name,
	}
	resourceStorageWithStringSize := parseResourceStorageWithStringSize(jsonStr, config.MapKey)
	if resourceStorageWithStringSize != nil {
		resourceStorageConfigValue.Value = &MetaResourceStorage{
			ResourceStorage: resourceStorageWithStringSize.ResourceStorage,
		}

		if resourceStorageWithStringSize.Size != "" {
			resourceStorageConfigValue.Value.Size = utils.ParseK8sResourceStorage(resourceStorageWithStringSize.Size)
		}
	} else {
		storageClass := ""
		storageAccessMode := ""
		var storageSize int64
		if config.StorageClassMapKey != "" {
			storageClass = gjson.Get(jsonStr, config.StorageClassMapKey).Str
		}
		if config.AccessModeMapKey != "" {
			storageAccessMode = gjson.Get(jsonStr, config.AccessModeMapKey).Str
		}
		if config.SizeMapKey != "" {
			storageSize = utils.ParseK8sResourceStorage(gjson.Get(jsonStr, config.SizeMapKey).Str)
		}
		if storageClass != "" || storageAccessMode != "" || storageSize != 0 {
			resourceStorageConfigValue.Value = &MetaResourceStorage{
				ResourceStorage: ResourceStorage{
					StorageClass: "silver",
					AccessModes:  []string{"ReadWriteOnce"},
				},
			}
			if storageClass != "" {
				resourceStorageConfigValue.Value.StorageClass = storageClass
			}
			if storageAccessMode != "" {
				resourceStorageConfigValue.Value.AccessModes = []string{storageAccessMode}
			}
			if storageSize != 0 {
				resourceStorageConfigValue.Value.Size = storageSize
			}
		}
	}

	return resourceStorageConfigValue
}

func parseResourceStorageWithStringSize(jsonStr, mapKey string) *MetaResourceStorageWithStringSize {
	rawMsg := gjson.Get(jsonStr, mapKey).Raw
	if rawMsg == "" {
		return nil
	}
	resourceStorage := &MetaResourceStorageWithStringSize{}
	err := json.Unmarshal([]byte(rawMsg), resourceStorage)
	if err != nil {
		klog.Warningf("failed to unmarshal %s : %s", rawMsg, err.Error())
		return nil
	}
	return resourceStorage
}

func ConvertResourceBinaryIntByUnit(i *int64, unit string) string {
	return strconv.FormatInt(*i, 10) + unit
}

func convertResourceDecimalFloat(f *float64) string {
	return fmt.Sprintf("%g", *f)
}
