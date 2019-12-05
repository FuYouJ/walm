package impl

import (
	"testing"
	"WarpCloud/walm/pkg/models/release"
	"github.com/stretchr/testify/assert"
)

func Test_convertPrettyParamsToMetainfoParams(t *testing.T) {
	testReplicas := int64(3)

	tests := []struct {
		metaInfo       *release.ChartMetaInfo
		prettyParams   *release.PrettyChartParams
		metaInfoParams *release.MetaInfoParams
		err            error
	}{
		{
			metaInfo: &release.ChartMetaInfo{
				ChartRoles: []*release.MetaRoleConfig{
					{
						Name: "zookeeper",
						RoleBaseConfig: &release.MetaRoleBaseConfig{
							Replicas: &release.MetaIntConfig{
								IntConfig: release.IntConfig{
									MetaInfoCommonConfig: release.MetaInfoCommonConfig{
										Variable:    "replicas",
										Description: "副本个数",
										Type:        "number",
									},
									DefaultValue: 0,
								},
							},
						},
					},
				},
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Type: "kvpair",
						},
						Name: "zoo.cfg",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				CommonConfig: release.CommonConfig{
					Roles: []*release.RoleConfig{
						{
							Name: "zookeeper",
							RoleBaseConfig: []*release.BaseConfig{
								{
									Name:             "replicas",
									Variable:         "replicas",
									DefaultValue:     3,
									ValueDescription: "副本个数",
									ValueType:        "number",
								},
							},
						},
					},
				},
				AdvanceConfig: []*release.BaseConfig{
					{
						Name:         "zoo.cfg",
						DefaultValue: map[string]interface{}{"test": "test"},
					},
				},
			},
			metaInfoParams: &release.MetaInfoParams{
				Roles: []*release.MetaRoleConfigValue{
					{
						Name: "zookeeper",
						RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
							Replicas: &testReplicas,
						},
					},
				},
				Params: []*release.MetaCommonConfigValue{
					{
						Name:  "zoo.cfg",
						Type:  "kvpair",
						Value: "{\"test\":\"test\"}",
					},
				},
			},
		},
	}

	for _, test := range tests {
		metainfoParams, err := convertPrettyParamsToMetainfoParams(test.metaInfo, test.prettyParams)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.metaInfoParams, metainfoParams)
	}
}

func Test_getCommonConfig(t *testing.T) {
	tests := []struct {
		metaCommonConfigs []*release.MetaCommonConfig
		baseConfig        *release.BaseConfig
		commonConfig      *release.MetaCommonConfig
	}{
		{
			metaCommonConfigs: []*release.MetaCommonConfig{
				{
					MetaInfoCommonConfig: release.MetaInfoCommonConfig{
						Type:     "kvpair",
						Variable: "Advance_Config.zoo_cfg",
					},
					Name: "zoo.cfg",
				},
			},
			baseConfig: &release.BaseConfig{
				Name: "zoo.cfg",
			},
			commonConfig: &release.MetaCommonConfig{
				MetaInfoCommonConfig: release.MetaInfoCommonConfig{
					Type:     "kvpair",
					Variable: "Advance_Config.zoo_cfg",
				},
				Name: "zoo.cfg",
			},
		},
		{
			metaCommonConfigs: []*release.MetaCommonConfig{
				{
					MetaInfoCommonConfig: release.MetaInfoCommonConfig{
						Type:     "kvpair",
						Variable: "Advance_Config.zoo_cfg",
					},
					Name: "zoo.cfg",
				},
			},
			baseConfig: &release.BaseConfig{
				Variable: "Advance_Config.zoo_cfg",
			},
			commonConfig: &release.MetaCommonConfig{
				MetaInfoCommonConfig: release.MetaInfoCommonConfig{
					Type:     "kvpair",
					Variable: "Advance_Config.zoo_cfg",
				},
				Name: "zoo.cfg",
			},
		},
		{
			metaCommonConfigs: []*release.MetaCommonConfig{
				{
					MetaInfoCommonConfig: release.MetaInfoCommonConfig{
						Type:     "kvpair",
						Variable: "Advance_Config.zoo_cfg",
					},
					Name: "zoo.cfg",
				},
			},
			baseConfig: &release.BaseConfig{
				Name: "notExisted",
			},
			commonConfig: nil,
		},
	}

	for _, test := range tests {
		commonConfig := getCommonConfig(test.metaCommonConfigs, test.baseConfig)
		assert.Equal(t, test.commonConfig, commonConfig)
	}
}

func Test_computeMetaRoleConfigValue(t *testing.T) {
	testReplicas := int64(3)
	testUseHostNetwork := true

	testLimitsGpu := float64(2)
	testRequestsGpu := float64(1)
	testLimitsCpu := float64(0.2)
	testRequestsCpu := float64(0.1)
	testLimitsMemory := int64(2048)
	testRequestsMemory := int64(1024)

	tests := []struct {
		metaRoleConfig      *release.MetaRoleConfig
		roleConfig          *release.RoleConfig
		metaRoleConfigValue *release.MetaRoleConfigValue
		err                 error
	}{
		{
			metaRoleConfig: &release.MetaRoleConfig{
				Name: "zookeeper",
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			roleConfig: &release.RoleConfig{
				Name: "zookeeper",
				RoleBaseConfig: []*release.BaseConfig{
					{
						Name:             "replicas",
						Variable:         "replicas",
						DefaultValue:     3,
						ValueDescription: "副本个数",
						ValueType:        "number",
					},
					{
						Name:             "useHostNetwork",
						Variable:         "use_host_network",
						DefaultValue:     true,
						ValueType:        "bool",
						ValueDescription: "是否使用主机网络",
					},
				},
				RoleResourceConfig: &release.ResourceConfig{
					GpuLimit:      2,
					GpuRequest:    1,
					CpuLimit:      0.2,
					CpuRequest:    0.1,
					MemoryLimit:   2048,
					MemoryRequest: 1024,
					ResourceStorageList: []release.ResourceStorageConfig{
						{
							Name:         "data",
							StorageClass: "silver",
							Size:         "100Gi",
							AccessModes: []string{
								"readwrite",
							},
							DiskReplicas: 2,
						},
					},
				},
			},
			metaRoleConfigValue: &release.MetaRoleConfigValue{
				Name: "zookeeper",
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Replicas:       &testReplicas,
					UseHostNetwork: &testUseHostNetwork,
				},
				RoleResourceConfigValue: &release.MetaResourceConfigValue{
					LimitsGpu:      &testLimitsGpu,
					RequestsGpu:    &testRequestsGpu,
					LimitsCpu:      &testLimitsCpu,
					RequestsCpu:    &testRequestsCpu,
					LimitsMemory:   &testLimitsMemory,
					RequestsMemory: &testRequestsMemory,
					StorageResources: []*release.MetaResourceStorageConfigValue{
						{
							Name: "data",
							Value: &release.MetaResourceStorage{
								ResourceStorage: release.ResourceStorage{
									DiskReplicas: 2,
									AccessModes: []string{
										"readwrite",
									},
									StorageClass: "silver",
								},
								Size: 100,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		metaRoleConfigValue, err := computeMetaRoleConfigValue(test.metaRoleConfig, test.roleConfig)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.metaRoleConfigValue, metaRoleConfigValue)
	}
}

func Test_fillMetaRoleBaseConfigValue(t *testing.T) {
	testReplicas := int64(3)
	testUseHostNetwork := true
	testPriority := int64(10)
	testImage := "zookeeper:transwarp-5.2"

	tests := []struct {
		metaRoleConfigValue *release.MetaRoleConfigValue
		metaRoleConfig      *release.MetaRoleConfig
		baseConfig          *release.BaseConfig
		result              *release.MetaRoleConfigValue
		err                 error
	}{
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Name:             "replicas",
				Variable:         "replicas",
				DefaultValue:     3,
				ValueDescription: "副本个数",
				ValueType:        "number",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Replicas: &testReplicas,
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Variable:         "replicas",
				DefaultValue:     3,
				ValueDescription: "副本个数",
				ValueType:        "number",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Replicas: &testReplicas,
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Variable:         "notExisted",
				DefaultValue:     3,
				ValueDescription: "副本个数",
				ValueType:        "number",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Name:             "useHostNetwork",
				Variable:         "use_host_network",
				DefaultValue:     true,
				ValueType:        "bool",
				ValueDescription: "是否使用主机网络",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					UseHostNetwork: &testUseHostNetwork,
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Variable:         "use_host_network",
				DefaultValue:     true,
				ValueType:        "bool",
				ValueDescription: "是否使用主机网络",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					UseHostNetwork: &testUseHostNetwork,
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Name:             "priority",
				Variable:         "priority",
				DefaultValue:     int64(10),
				ValueType:        "number",
				ValueDescription: "优先级",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Priority: &testPriority,
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Variable:         "priority",
				DefaultValue:     int64(10),
				ValueType:        "number",
				ValueDescription: "优先级",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Priority: &testPriority,
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
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
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Env: []release.MetaEnv{
						{
							Name:  "test",
							Value: "test",
						},
					},
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
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
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Env: []release.MetaEnv{
						{
							Name:  "test",
							Value: "test",
						},
					},
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Name:     "envMap",
				Variable: "env_map",
				DefaultValue: map[string]string{
					"test": "test",
				},
				ValueType:        "map",
				ValueDescription: "env map",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					EnvMap: map[string]string{
						"test": "test",
					},
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Variable: "env_map",
				DefaultValue: map[string]string{
					"test": "test",
				},
				ValueType:        "map",
				ValueDescription: "env map",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					EnvMap: map[string]string{
						"test": "test",
					},
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Name:             "image",
				Variable:         "image",
				DefaultValue:     "zookeeper:transwarp-5.2",
				ValueType:        "string",
				ValueDescription: "镜像",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Image: &testImage,
				},
			},
		},
		{
			metaRoleConfigValue: &release.MetaRoleConfigValue{},
			metaRoleConfig: &release.MetaRoleConfig{
				RoleBaseConfig: &release.MetaRoleBaseConfig{
					Replicas: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "replicas",
								Description: "副本个数",
								Type:        "number",
							},
						},
					},
					EnvMap: &release.MetaEnvMapConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_map",
							Description: "env map",
							Type:        "map",
						},
					},
					UseHostNetwork: &release.MetaBoolConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "use_host_network",
							Description: "是否使用主机网络",
							Type:        "bool",
						},
					},
					Priority: &release.MetaIntConfig{
						IntConfig: release.IntConfig{
							MetaInfoCommonConfig: release.MetaInfoCommonConfig{
								Variable:    "priority",
								Description: "优先级",
								Type:        "number",
							},
						},
					},
					Env: &release.MetaEnvConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "env_list",
							Description: "env list",
							Type:        "list",
						},
					},
					Image: &release.MetaStringConfig{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "image",
							Description: "镜像",
							Type:        "string",
						},
					},
				},
			},
			baseConfig: &release.BaseConfig{
				Variable:         "image",
				DefaultValue:     "zookeeper:transwarp-5.2",
				ValueType:        "string",
				ValueDescription: "镜像",
			},
			result: &release.MetaRoleConfigValue{
				RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
					Image: &testImage,
				},
			},
		},
	}

	for _, test := range tests {
		err := fillMetaRoleBaseConfigValue(test.metaRoleConfigValue, test.metaRoleConfig, test.baseConfig)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.result, test.metaRoleConfigValue)
	}
}

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
				CpuRequest:    0.1,
				CpuLimit:      0.2,
				GpuRequest:    1,
				GpuLimit:      2,
				MemoryRequest: 1024,
				MemoryLimit:   2048,
				ResourceStorageList: []release.ResourceStorageConfig{
					{
						Name:         "data",
						Size:         "30Gi",
						DiskReplicas: 3,
						AccessModes:  []string{"readwrite"},
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

func Test_convertMetaInfoParamsToPrettyParams(t *testing.T) {
	testReplicas := int64(1)
	testUseHostNetwork := true
	testPriority := int64(10)
	testImage := "zookeeper:transwarp-5.2"

	testLimitsGpu := float64(2)
	testRequestsGpu := float64(1)
	testLimitsCpu := float64(0.2)
	testRequestsCpu := float64(0.1)
	testLimitsMemory := int64(2048)
	testRequestsMemory := int64(1024)

	tests := []struct {
		metaInfo       *release.ChartMetaInfo
		metaInfoParams *release.MetaInfoParams
		prettyParams   *release.PrettyChartParams
	}{
		{
			metaInfo: &release.ChartMetaInfo{
				ChartRoles: []*release.MetaRoleConfig{
					{
						Name: "zookeeper",
						RoleBaseConfig: &release.MetaRoleBaseConfig{
							Replicas: &release.MetaIntConfig{
								IntConfig: release.IntConfig{
									MetaInfoCommonConfig: release.MetaInfoCommonConfig{
										Variable:    "replicas",
										Description: "副本个数",
										Type:        "number",
									},
								},
							},
							UseHostNetwork: &release.MetaBoolConfig{
								MetaInfoCommonConfig: release.MetaInfoCommonConfig{
									Variable:    "use_host_network",
									Description: "是否使用主机网络",
									Type:        "bool",
								},
							},
							Priority: &release.MetaIntConfig{
								IntConfig: release.IntConfig{
									MetaInfoCommonConfig: release.MetaInfoCommonConfig{
										Variable:    "priority",
										Description: "优先级",
										Type:        "number",
									},
								},
							},
							Env: &release.MetaEnvConfig{
								MetaInfoCommonConfig: release.MetaInfoCommonConfig{
									Variable:    "env_list",
									Description: "env list",
									Type:        "list",
								},
							},
							EnvMap: &release.MetaEnvMapConfig{
								MetaInfoCommonConfig: release.MetaInfoCommonConfig{
									Variable:    "env_map",
									Description: "env map",
									Type:        "map",
								},
							},
							Image: &release.MetaStringConfig{
								MetaInfoCommonConfig: release.MetaInfoCommonConfig{
									Variable:    "image",
									Description: "镜像",
									Type:        "string",
								},
							},
						},
					},
				},
				ChartParams: []*release.MetaCommonConfig{
					{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "Advance_Config.zoo_cfg",
							Type:        "kvpair",
							Description: "zoo cfg",
						},
						Name: "zoo.cfg",
					},
					{
						MetaInfoCommonConfig: release.MetaInfoCommonConfig{
							Variable:    "trun.test",
							Type:        "string",
							Description: "test cfg",
						},
						Name: "test",
						VariableType: release.TranswarpBundleConfig,
					},
				},
			},
			metaInfoParams: &release.MetaInfoParams{
				Roles: []*release.MetaRoleConfigValue{
					{
						Name: "zookeeper",
						RoleBaseConfigValue: &release.MetaRoleBaseConfigValue{
							Replicas:       &testReplicas,
							UseHostNetwork: &testUseHostNetwork,
							Priority:       &testPriority,
							Env: []release.MetaEnv{
								{
									Name:  "test",
									Value: "test",
								},
							},
							EnvMap: map[string]string{
								"test": "test",
							},
							Image: &testImage,
						},
						RoleResourceConfigValue: &release.MetaResourceConfigValue{
							LimitsGpu:      &testLimitsGpu,
							RequestsGpu:    &testRequestsGpu,
							LimitsCpu:      &testLimitsCpu,
							RequestsCpu:    &testRequestsCpu,
							LimitsMemory:   &testLimitsMemory,
							RequestsMemory: &testRequestsMemory,
							StorageResources: []*release.MetaResourceStorageConfigValue{
								{
									Name: "data",
									Value: &release.MetaResourceStorage{
										ResourceStorage: release.ResourceStorage{
											DiskReplicas: 2,
											AccessModes: []string{
												"readwrite",
											},
											StorageClass: "silver",
										},
										Size: 100,
									},
								},
							},
						},
					},
				},
				Params: []*release.MetaCommonConfigValue{
					{
						Name:  "zoo.cfg",
						Value: "{\"test\":\"test\"}",
					},
					{
						Name:  "test",
						Value: "\"test\"",
					},
				},
			},
			prettyParams: &release.PrettyChartParams{
				CommonConfig: release.CommonConfig{
					Roles: []*release.RoleConfig{
						{
							Name: "zookeeper",
							RoleBaseConfig: []*release.BaseConfig{
								{
									Name:             "replicas",
									Variable:         "replicas",
									DefaultValue:     int64(1),
									ValueDescription: "副本个数",
									ValueType:        "number",
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
									DefaultValue:     int64(10),
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
							RoleResourceConfig: &release.ResourceConfig{
								GpuLimit:      2,
								GpuRequest:    1,
								CpuLimit:      0.2,
								CpuRequest:    0.1,
								MemoryLimit:   2048,
								MemoryRequest: 1024,
								ResourceStorageList: []release.ResourceStorageConfig{
									{
										Name:         "data",
										StorageClass: "silver",
										Size:         "100Gi",
										AccessModes: []string{
											"readwrite",
										},
										DiskReplicas: 2,
									},
								},
							},
						},
					},
				},
				AdvanceConfig: []*release.BaseConfig{
					{
						Name:             "zoo.cfg",
						Variable:         "Advance_Config.zoo_cfg",
						ValueDescription: "zoo cfg",
						ValueType:        "kvpair",
						DefaultValue:     map[string]interface{}{"test": "test"},
					},
				},
				TranswarpBaseConfig: []*release.BaseConfig{
					{
						Name:             "test",
						Variable:         "trun.test",
						ValueDescription: "test cfg",
						ValueType:        "string",
						DefaultValue:     "test",
					},
				},
			},
		},
	}

	for _, test := range tests {
		prettyParams := convertMetaInfoParamsToPrettyParams(test.metaInfo, test.metaInfoParams)
		assert.Equal(t, test.prettyParams, prettyParams)
	}

}
