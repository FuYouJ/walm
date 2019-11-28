package impl

import (
	"testing"
	"WarpCloud/walm/pkg/models/release"
	"github.com/stretchr/testify/assert"
)

//func Test_convertPrettyParamsToMetainfoParams(t *testing.T) {
//	tests := []struct {
//		prettyParams   *release.PrettyChartParams
//		metaInfoParams *release.MetaInfoParams
//	}{
//		{
//			prettyParams: &release.PrettyChartParams{
//				CommonConfig: release.CommonConfig{
//					Roles: []*release.RoleConfig{
//						{
//							Name:        "zookeeper",
//							Description: "zookeeper服务",
//						},
//					},
//				},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		metainfoParams := convertPrettyParamsToMetainfoParams(test.prettyParams)
//		assert.Equal(t, test.metaInfoParams, metainfoParams)
//	}
//}

func Test_convertMetainfoToPrettyParams(t *testing.T) {
	tests := []struct {
		metaInfo     *release.ChartMetaInfo
		prettyParams *release.PrettyChartParams
	}{
		{
			metaInfo: &release.ChartMetaInfo{
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.NewMetaInfoCommonConfig("", "test-ds",
							"string", "", false),
						Name:         "test-ns",
						DefaultValue: "\"test-value\"",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				AdvanceConfig: []*release.BaseConfig{
					{
						Name:             "test-ns",
						ValueDescription: "test-ds",
						DefaultValue:     "test-value",
						ValueType:        "string",
					},
				},
			},
		},
		{
			metaInfo: &release.ChartMetaInfo{
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.NewMetaInfoCommonConfig("", "test-ds",
							"string", "", false),
						Name: "test-ns",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				AdvanceConfig: []*release.BaseConfig{
					{
						Name:             "test-ns",
						ValueDescription: "test-ds",
						ValueType:        "string",
					},
				},
			},
		},
		{
			metaInfo: &release.ChartMetaInfo{
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.NewMetaInfoCommonConfig("", "test-ds",
							"string", "Advanced.test-ns", false),
						Name:         "test-ns",
						DefaultValue: "\"test-value\"",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				AdvanceConfig: []*release.BaseConfig{
					{
						Name:             "test-ns",
						Variable:         "Advanced.test-ns",
						ValueDescription: "test-ds",
						DefaultValue:     "test-value",
						ValueType:        "string",
					},
				},
			},
		},
		{
			metaInfo: &release.ChartMetaInfo{
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.NewMetaInfoCommonConfig("", "test-ds",
							"string", "Advanced.test-ns", false),
						Name:         "test-ns",
						DefaultValue: "\"test-value\"",
						VariableType: "advanceConfig",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				AdvanceConfig: []*release.BaseConfig{
					{
						Name:             "test-ns",
						Variable:         "Advanced.test-ns",
						ValueDescription: "test-ds",
						DefaultValue:     "test-value",
						ValueType:        "string",
					},
				},
			},
		},
		{
			metaInfo: &release.ChartMetaInfo{
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.NewMetaInfoCommonConfig("", "test-ds",
							"string", "Advanced.test-ns", false),
						Name:         "test-ns",
						DefaultValue: "\"test-value\"",
						VariableType: "transwarpBundleConfig",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				TranswarpBaseConfig: []*release.BaseConfig{
					{
						Name:             "test-ns",
						Variable:         "Advanced.test-ns",
						ValueDescription: "test-ds",
						DefaultValue:     "test-value",
						ValueType:        "string",
					},
				},
			},
		},
		{
			metaInfo: &release.ChartMetaInfo{
				ChartRoles: []*release.MetaRoleConfig{
					{
						Name:        "zookeeper",
						Description: "zookeeper service",
						RoleBaseConfig: &release.MetaRoleBaseConfig{
							Replicas: &release.MetaIntConfig{
								IntConfig: release.IntConfig{
									MetaInfoCommonConfig: release.MetaInfoCommonConfig{
										Variable:    "replicas",
										Description: "副本个数",
										Type:        "number",
									},
									DefaultValue: 3,
								},
							},
						},
						RoleResourceConfig: &release.MetaResourceConfig{
							RequestsCpu: &release.MetaResourceCpuConfig{
								FloatConfig: release.FloatConfig{
									DefaultValue: 0.1,
								},
							},
						},
					},
				},
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "Advanced.test-ns",
							Description: "test-ds",
							Type:        "string",
						},
						Name:         "test-ns",
						DefaultValue: "\"test-value\"",
						VariableType: "transwarpBundleConfig",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				CommonConfig: release.CommonConfig{
					Roles: []*release.RoleConfig{
						{
							Name:        "zookeeper",
							Description: "zookeeper service",
							RoleBaseConfig: []*release.BaseConfig{
								{
									Name:             "replicas",
									DefaultValue:     int64(3),
									Variable:         "replicas",
									ValueDescription: "副本个数",
									ValueType:        "number",
								},
							},
							RoleResourceConfig: &release.ResourceConfig{
								CpuRequest: float64(0.1),
							},
						},
					},
				},
				TranswarpBaseConfig: []*release.BaseConfig{
					{
						Name:             "test-ns",
						Variable:         "Advanced.test-ns",
						ValueDescription: "test-ds",
						DefaultValue:     "test-value",
						ValueType:        "string",
					},
				},
			},
		},
	}

	for _, test := range tests {
		prettyParams := convertMetainfoToPrettyParams(test.metaInfo)
		assert.Equal(t, test.prettyParams, prettyParams)
	}
}

func Test_convertMetaRoleBaseConfigToBaseConfigs(t *testing.T) {
	tests := []struct {
		metaRoleBaseConfig *release.MetaRoleBaseConfig
		baseConfigs        []*release.BaseConfig
	}{
		{
			metaRoleBaseConfig: &release.MetaRoleBaseConfig{
				Replicas: &release.MetaIntConfig{
					IntConfig: release.IntConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "replicas",
							Description: "副本个数",
							Type:        "number",
						},
						DefaultValue: 3,
					},
				},
				UseHostNetwork: &release.MetaBoolConfig{
					MetaInfoCommonConfig: release.MetaInfoCommonConfig{
						Variable:    "use_host_network",
						Description: "是否使用主机网络",
						Type:        "bool",
					},
					DefaultValue: true,
				},
				Priority: &release.MetaIntConfig{
					IntConfig: release.IntConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "priority",
							Description: "优先级",
							Type:        "number",
						},
						DefaultValue: 0,
					},
				},
				Env: &release.MetaEnvConfig{
					MetaInfoCommonConfig: release.MetaInfoCommonConfig{
						Variable:    "env_list",
						Description: "env list",
						Type:        "list",
					},
					DefaultValue: []release.MetaEnv{
						{
							Name:  "test",
							Value: "test",
						},
					},
				},
				EnvMap: &release.MetaEnvMapConfig{
					MetaInfoCommonConfig: release.MetaInfoCommonConfig{
						Variable:    "env_map",
						Description: "env map",
						Type:        "map",
					},
					DefaultValue: map[string]string{
						"test": "test",
					},
				},
				Image: &release.MetaStringConfig{
					MetaInfoCommonConfig: release.MetaInfoCommonConfig{
						Variable:    "image",
						Description: "镜像",
						Type:        "string",
					},
					DefaultValue: "zookeeper:transwarp-5.2",
				},
			},
			baseConfigs: []*release.BaseConfig{
				{
					Name:             "replicas",
					Variable:         "replicas",
					DefaultValue:     int64(3),
					ValueType:        "number",
					ValueDescription: "副本个数",
				},
				{
					Name:             "useHostNetwork",
					Variable:         "use_host_network",
					DefaultValue:     true,
					ValueType:        "bool",
					ValueDescription: "是否使用主机网络",
				},
				{
					Name:             "priority",
					Variable:         "priority",
					DefaultValue:     int64(0),
					ValueType:        "number",
					ValueDescription: "优先级",
				},
				{
					Name:     "envList",
					Variable: "env_list",
					DefaultValue: []release.MetaEnv{
						{
							Name:  "test",
							Value: "test",
						},
					},
					ValueType:        "list",
					ValueDescription: "env list",
				},
				{
					Name:     "envMap",
					Variable: "env_map",
					DefaultValue: map[string]string{
						"test": "test",
					},
					ValueType:        "map",
					ValueDescription: "env map",
				},
				{
					Name:             "image",
					Variable:         "image",
					DefaultValue:     "zookeeper:transwarp-5.2",
					ValueType:        "string",
					ValueDescription: "镜像",
				},
			},
		},
	}

	for _, test := range tests {
		baseConfigs := convertMetaRoleBaseConfigToBaseConfigs(test.metaRoleBaseConfig)
		assert.ElementsMatch(t, test.baseConfigs, baseConfigs)
	}
}

func Test_convertMetaResourceConfigToResourceConfig(t *testing.T) {
	tests := []struct {
		metaResourceConfig *release.MetaResourceConfig
		resourceConfig     *release.ResourceConfig
	}{
		{
			metaResourceConfig: &release.MetaResourceConfig{
				RequestsCpu: &release.MetaResourceCpuConfig{
					FloatConfig: release.FloatConfig{
						DefaultValue: 0.1,
					},
				},
				LimitsCpu: &release.MetaResourceCpuConfig{
					FloatConfig: release.FloatConfig{
						DefaultValue: 0.2,
					},
				},
				RequestsGpu: &release.MetaResourceCpuConfig{
					FloatConfig: release.FloatConfig{
						DefaultValue: 1,
					},
				},
				LimitsGpu: &release.MetaResourceCpuConfig{
					FloatConfig: release.FloatConfig{
						DefaultValue: 2,
					},
				},
				RequestsMemory: &release.MetaResourceMemoryConfig{
					IntConfig: release.IntConfig{
						DefaultValue: 1024,
					},
				},
				LimitsMemory: &release.MetaResourceMemoryConfig{
					IntConfig: release.IntConfig{
						DefaultValue: 2048,
					},
				},
				StorageResources: []*release.MetaResourceStorageConfig{
					{
						Name: "data",
						DefaultValue: &release.MetaResourceStorage{
							ResourceStorage: release.ResourceStorage{
								StorageClass: "silver",
								AccessModes:  []string{"readwrite"},
								DiskReplicas: 3,
							},
							Size: 30,
						},
					},
				},
			},
			resourceConfig: &release.ResourceConfig{
				CpuRequest: 0.1,
				CpuLimit: 0.2,
				GpuRequest: 1,
				GpuLimit: 2,
				MemoryRequest: 1024,
				MemoryLimit: 2048,
				ResourceStorageList: []release.ResourceStorageConfig{
					{
						Name: "data",
						Size: "30Gi",
						DiskReplicas: 3,
						AccessModes: []string{"readwrite"},
						StorageClass: "silver",
					},
				},
			},
		},
	}

	for _, test := range tests {
		resourceConfig := convertMetaResourceConfigToResourceConfig(test.metaResourceConfig)
		assert.Equal(t, test.resourceConfig, resourceConfig)
	}
}
