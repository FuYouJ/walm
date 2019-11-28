package release

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"encoding/json"
)

func TestMetaInfoParams_BuildConfigValues(t *testing.T) {
	image := "test-image"
	replicas := int64(2)
	useHostNetwork := true
	priority := int64(1)

	requestCpu := float64(0.1)
	limitsCpu := float64(0.2)
	requestsMem := int64(200)
	limitsMem := int64(400)
	requestsGpu := float64(1)
	limitsGpu := float64(2)

	tests := []struct {
		params       *MetaInfoParams
		metaInfo     *ChartMetaInfo
		configValues string
		err          error
	}{
		{
			params: &MetaInfoParams{
				Roles: []*MetaRoleConfigValue{
					{
						Name: "test-role",
						RoleBaseConfigValue: &MetaRoleBaseConfigValue{
							Image:          &image,
							Replicas:       &replicas,
							UseHostNetwork: &useHostNetwork,
							Priority:       &priority,
							Env: []MetaEnv{
								{
									Name:  "test-key",
									Value: "test-value",
								},
							},
							Others: []*MetaCommonConfigValue{
								{
									Name:  "test-other",
									Type:  "string",
									Value: "\"test-other-value\"",
								},
							},
						},
						RoleResourceConfigValue: &MetaResourceConfigValue{
							RequestsCpu:    &requestCpu,
							LimitsCpu:      &limitsCpu,
							RequestsMemory: &requestsMem,
							LimitsMemory:   &limitsMem,
							RequestsGpu:    &requestsGpu,
							LimitsGpu:      &limitsGpu,
							StorageResources: []*MetaResourceStorageConfigValue{
								{
									Name: "test-storage",
									Value: &MetaResourceStorage{
										ResourceStorage: ResourceStorage{
											AccessModes:  []string{"ReadOnly"},
											StorageClass: "silver",
										},
										Size: 100,
									},
								},
							},
						},
					},
				},
				Params: []*MetaCommonConfigValue{
					{
						Name:  "test-params",
						Type:  "string",
						Value: "\"test-params-value\"",
					},
				},
			},
			metaInfo: &ChartMetaInfo{
				ChartRoles: []*MetaRoleConfig{
					{
						Name: "test-role",
						RoleBaseConfig: &MetaRoleBaseConfig{
							Image: &MetaStringConfig{
								MetaInfoCommonConfig: NewMetaInfoCommonConfig("image.application.image", "", "", "", false),
							},
							Env: &MetaEnvConfig{
								MetaInfoCommonConfig: NewMetaInfoCommonConfig("envs", "", "", "", false),
							},
							Priority: &MetaIntConfig{
								IntConfig: IntConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("priority", "", "", "", false),
								},
							},
							UseHostNetwork: &MetaBoolConfig{
								MetaInfoCommonConfig: NewMetaInfoCommonConfig("useHostNetwork", "", "", "", false),
							},
							Replicas: &MetaIntConfig{
								IntConfig: IntConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("replicas", "", "", "", false),
								},
							},
							Others: []*MetaCommonConfig{
								{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("test-other", "", "", "", false),
									Name:   "test-other",
								},
							},
						},
						RoleResourceConfig: &MetaResourceConfig{
							RequestsCpu: &MetaResourceCpuConfig{
								FloatConfig: FloatConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("resources.requestsCpu", "", "", "", false),
								},
							},
							LimitsCpu: &MetaResourceCpuConfig{
								FloatConfig: FloatConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("resources.limitsCpu", "", "", "", false),
								},
							},
							RequestsMemory: &MetaResourceMemoryConfig{
								IntConfig: IntConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("resources.requestsMem", "", "", "", false),
								},
							},
							LimitsMemory: &MetaResourceMemoryConfig{
								IntConfig: IntConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("resources.LimitsMem", "", "", "", false),
								},
							},
							RequestsGpu: &MetaResourceCpuConfig{
								FloatConfig: FloatConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("resources.requestsGpu", "", "", "", false),
								},
							},
							LimitsGpu: &MetaResourceCpuConfig{
								FloatConfig: FloatConfig{
									MetaInfoCommonConfig: NewMetaInfoCommonConfig("resources.limitsGpu", "", "", "", false),
								},
							},
							StorageResources: []*MetaResourceStorageConfig{
								{
									Name:   "test-storage",
									MapKey: "storage",
								},
							},
						},
					},
				},
				ChartParams: []*MetaCommonConfig{
					{
						MetaInfoCommonConfig: NewMetaInfoCommonConfig("image.java.command", "", "", "", false),
						Name:   "test-params",
					},
				},
			},
			configValues: "{\"envs\":[{\"name\":\"test-key\",\"value\":\"test-value\"}],\"image\":{\"application\":{\"image\":\"test-image\"},\"java\":{\"command\":\"test-params-value\"}},\"priority\":1,\"replicas\":2,\"resources\":{\"LimitsMem\":\"400Mi\",\"limitsCpu\":\"0.2\",\"limitsGpu\":\"2\",\"requestsCpu\":\"0.1\",\"requestsGpu\":\"1\",\"requestsMem\":\"200Mi\"},\"storage\":{\"accessModes\":[\"ReadOnly\"],\"diskReplicas\":0,\"size\":\"100Gi\",\"storageClass\":\"silver\"},\"test-other\":\"test-other-value\",\"useHostNetwork\":true}",
			err:          nil,
		},
		{
			params: &MetaInfoParams{
				Roles: []*MetaRoleConfigValue{
					{
						Name: "test-role",
						RoleResourceConfigValue: &MetaResourceConfigValue{
							StorageResources: []*MetaResourceStorageConfigValue{
								{
									Name: "test-storage",
									Value: &MetaResourceStorage{
										ResourceStorage: ResourceStorage{
											AccessModes:  []string{"ReadOnly"},
											StorageClass: "silver",
										},
										Size: 100,
									},
								},
							},
						},
					},
				},
			},
			metaInfo: &ChartMetaInfo{
				ChartRoles: []*MetaRoleConfig{
					{
						Name: "test-role",
						RoleResourceConfig: &MetaResourceConfig{
							StorageResources: []*MetaResourceStorageConfig{
								{
									Name:   "test-storage",
									MapKey: "storage",
									AccessModeMapKey: "storageex.accessModeMapKey",
									StorageClassMapKey: "storageex.storageClassMapKey",
									SizeMapKey: "storageex.sizeMapKey",
								},
							},
						},
					},
				},
			},
			configValues: "{\"storage\":{\"accessModes\":[\"ReadOnly\"],\"diskReplicas\":0,\"size\":\"100Gi\",\"storageClass\":\"silver\"},\"storageex\":{\"accessModeMapKey\":\"ReadOnly\",\"sizeMapKey\":\"100Gi\",\"storageClassMapKey\":\"silver\"}}",
			err:          nil,
		},
	}

	for _, test := range tests {
		configValues, err := test.params.BuildConfigValues(test.metaInfo)
		assert.IsType(t, test.err, err)

		configValuesStr, err := json.Marshal(configValues)
		assert.IsType(t, nil, err)
		assert.Equal(t, test.configValues, string(configValuesStr))
	}
}
