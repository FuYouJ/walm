package release

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChartMetaInfo_CheckMetainfoValidate(t *testing.T) {
	tests := []struct {
		chartMetaInfo *ChartMetaInfo
		configMaps    []*MetaConfigTestSet
		err           error
	}{
		{
			chartMetaInfo: &ChartMetaInfo{
				FriendlyName:          "Zookeeper",
				Categories:            []string{"Transwarp Hub"},
				ChartDependenciesInfo: nil,
				ChartRoles: []*MetaRoleConfig{
					{
						Name:        "zookeeper",
						Description: "zookeeper",
						Type:        "container",
						RoleBaseConfig: &MetaRoleBaseConfig{
							Image: &MetaStringConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey:      "appConfig.zookeeper.containers.zookeeper.image",
									Description: "镜像",
									Type:        "string",
									Required:    true,
								},
							},
							Priority: &MetaIntConfig{IntConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey:      "appConfig.zookeeper.priority",
									Description: "优先级",
									Type:        "string",
									Required:    true,
								},
							}},
							Replicas: &MetaIntConfig{IntConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey:      "appConfig.zookeeper.replicas",
									Description: "副本个数",
									Type:        "number",
									Required:    true,
								},
							}},
							Env: nil,
							EnvMap: &MetaEnvMapConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey:      "appConfig.zookeeper.containers.zookeeper.envMap",
									Description: "额外环境变量",
									Type:        "envMap",
									Required:    true,
								},
							},
							UseHostNetwork: &MetaBoolConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey:      "appConfig.zookeeper.use_host_network",
									Description: "是否使用主机网络",
									Type:        "boolean",
									Required:    false,
								},
							},
							Others: nil,
						},
						RoleResourceConfig: &MetaResourceConfig{
							LimitsMemory: &MetaResourceMemoryConfig{IntConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey: "appConfig.zookeeper.containers.zookeeper.resources.memory_limit",
								},
							}},
							LimitsCpu: &MetaResourceCpuConfig{FloatConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey: "appConfig.zookeeper.containers.zookeeper.resources.cpu_limit",
								},
							}},
							LimitsGpu: &MetaResourceCpuConfig{FloatConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey: "appConfig.zookeeper.containers.zookeeper.resources.gpu_limit",
								},
							}},
							RequestsMemory: &MetaResourceMemoryConfig{IntConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey: "appConfig.zookeeper.containers.zookeeper.resources.memory_request",
								},
							}},
							RequestsCpu: &MetaResourceCpuConfig{FloatConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey: "appConfig.zookeeper.containers.zookeeper.resources.cpu_request",
								},
							}},
							RequestsGpu: &MetaResourceCpuConfig{FloatConfig{
								MetaInfoCommonConfig: MetaInfoCommonConfig{
									MapKey: "appConfig.zookeeper.containers.zookeeper.resources.gpu_request",
								},
							}},
							StorageResources: []*MetaResourceStorageConfig{
								{
									Name:        "data",
									MapKey:      "appConfig.zookeeper.containers.zookeeper.resources.storage.data",
									Description: "zookeeper数据目录配置",
									Type:        "storagePVCType",
									Required:    false,
								},
							},
						},
						RoleHealthCheckConfig: nil,
					},
				},
				ChartParams: []*MetaCommonConfig{
					{
						MetaInfoCommonConfig: MetaInfoCommonConfig{
							MapKey:      "advanceConfig.zookeeper",
							Description: "zookeeper 通用配置",
							Type:        "kvPair",
							Required:    true,
						},
						Name: "zookeeper-common-config",
					},
					{
						MetaInfoCommonConfig: MetaInfoCommonConfig{
							MapKey:      "advanceConfig.zoo_cfg",
							Description: "/etc/zookeeper/conf/zoo.cfg 额外配置键值对",
							Type:        "kvPair",
							Required:    true,
						},
						Name: "zoo.cfg",
					},
				},
				Plugins:           nil,
				CustomChartParams: nil,
			},

			configMaps: []*MetaConfigTestSet{
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.image",
					Type:     "string",
					Required: true,
				},
				{
					MapKey:   "appConfig.zookeeper.replicas",
					Type:     "int",
					Required: true,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.envMap",
					Type:     "envMap",
					Required: true,
				},
				{
					MapKey:   "appConfig.zookeeper.use_host_network",
					Type:     "boolean",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.priority",
					Type:     "int",
					Required: true,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.cpu_limit",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.gpu_limit",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.memory_limit",
					Type:     "string",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.cpu_request",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.gpu_request",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.memory_request",
					Type:     "string",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.storage.data",
					Type:     "storagePVCType",
					Required: false,
				},
				{
					MapKey:   "advanceConfig.zookeeper",
					Type:     "kvPair",
					Required: true,
				},
				{
					MapKey:   "advanceConfig.zoo_cfg",
					Type:     "kvPair",
					Required: true,
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		configMaps, err := test.chartMetaInfo.CheckMetainfoValidate()
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.configMaps, configMaps)
	}
}

func TestChartMetaInfo_CheckParamsInValues(t *testing.T) {
	tests := []struct {
		chartMetaInfo *ChartMetaInfo
		valuesStr     string
		configSets    []*MetaConfigTestSet
		err           error
	}{
		{
			chartMetaInfo: &ChartMetaInfo{
			},
			valuesStr: `{"advanceConfig":{"security":{"auth_type":"none"},"zoo_cfg":{"autopurge.purgeInterval":5,"autopurge.snapRetainCount":10,"initLimit":10,"maxClientCnxns":0,"syncLimit":5,"tickTime":9000},"zookeeper":{"zookeeper.client.port":2181,"zookeeper.jmxremote.port":9911,"zookeeper.leader.elect.port":3888,"zookeeper.peer.communicate.port":2888}},"appConfig":{"zookeeper":{"containers":{"zookeeper":{"command":["/boot/entrypoint.sh"],"envMap":{},"image":"172.16.1.99/gold/zookeeper:transwarp-6.1","livenessProbe":{"fmtContent":"exec:\n  command:\n  - /bin/bash\n  - -c\n  - echo ruok|nc localhost {{ index .advanceConfig.zookeeper \"zookeeper.client.port\" }} \u003e /dev/null \u0026\u0026 echo ok\n  periodSeconds: 30\n  initialDelaySeconds: 60\n","type":"gotmpl"},"readinessProbe":{"fmtContent":"exec:\n  command:\n  - /bin/bash\n  - -c\n  - echo ruok|nc localhost {{ index .advanceConfig.zookeeper \"zookeeper.client.port\" }} \u003e /dev/null \u0026\u0026 echo ok\n  periodSeconds: 30\n  initialDelaySeconds: 60\n","type":"gotmpl"},"resources":{"cpu_limit":2,"cpu_request":0.5,"memory_limit":"4Gi","memory_request":"1Gi","storage":{"data":{"accessMode":"ReadWriteOnce","limit":{},"mountPath":"/var/transwarp","size":"100Gi","storageClass":"silver","type":"pvc"}}}}},"priority":0,"replicas":3,"type":"StatefulSet","update_strategy_configs":{"type":"Recreate"},"use_host_network":false}},"transwarpConfig":{"transwarpLicenseAddress":"","transwarpMetrics":{"enable":true}}}`,
			configSets: []*MetaConfigTestSet{
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.image",
					Type:     "string",
					Required: true,
				},
				{
					MapKey:   "appConfig.zookeeper.priority",
					Type:     "int",
					Required: true,
				},
				{
					MapKey:   "appConfig.zookeeper.replicas",
					Type:     "int",
					Required: true,
				},
				{
					MapKey:   "appConfig.zookeeper.use_host_network",
					Type:     "boolean",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.cpu_limit",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.memory_limit",
					Type:     "string",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.gpu_limit",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.memory_request",
					Type:     "string",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.cpu_request",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.gpu_request",
					Type:     "float",
					Required: false,
				},
				{
					MapKey:   "appConfig.zookeeper.containers.zookeeper.resources.storage.data",
					Type:     "storagePVCType",
					Required: false,
				},
				{
					MapKey:   "advanceConfig.zookeeper",
					Type:     "kvPair",
					Required: true,
				},
				{
					MapKey:   "advanceConfig.zoo_cfg",
					Type:     "kvPair",
					Required: true,
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		err := test.chartMetaInfo.CheckParamsInValues(test.valuesStr, test.configSets)
		assert.IsType(t, test.err, err)
	}
}
