package impl

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"WarpCloud/walm/pkg/models/release"
	"github.com/ghodss/yaml"
	"k8s.io/klog"
	"WarpCloud/walm/pkg/util"
)

func Test_processPrettyParams(t *testing.T) {
	tests := []struct {
		prettyParamsStr string
		configValues    map[string]interface{}
		result          map[string]interface{}
	}{
		{
			prettyParamsStr: hdfsPrettyParams,
			configValues:    map[string]interface{}{},
			result: map[string]interface{}{
				"App": map[string]interface{}{
					"hdfsnamenode": map[string]interface{}{
						"resources": map[string]interface{}{
							"cpu_limit":   2,
							"gpu_request": 0,
							"gpu_limit":   0,
							"storage": map[string]interface{}{
								"data": map[string]interface{}{
									"storageClass":  "silver",
									"size":          "100Gi",
									"accessModes":   []string{"ReadWriteOnce"},
									"disk_replicas": 0,
								},
								"log": map[string]interface{}{
									"size":         "20Gi",
									"accessMode":   "ReadWriteOnce",
									"storageClass": "silver",
									"disk_replicas": 0,
								},
							},
							"memory_request": 4,
							"memory_limit":   8,
							"cpu_request":    1,
						},
						"image":            "172.16.1.99/gold/hdfs:transwarp-5.2",
						"priority":         10,
						"replicas":         2,
						"env_list":         []interface{}{},
						"use_host_network": false,
					},
					"hdfszkfc": map[string]interface{}{
						"resources": map[string]interface{}{
							"memory_request": 0.5,
							"memory_limit":   1,
							"cpu_request":    0.1,
							"cpu_limit":      0.5,
							"gpu_request":    0,
							"gpu_limit":      0,
						},
						"image":    "172.16.1.99/transwarp1111111111/hdfs:transwarp-5.2",
						"env_list": []interface{}{},
					},
					"hdfsdatanode": map[string]interface{}{
						"use_host_network": false,
						"resources": map[string]interface{}{
							"cpu_request": 0.5,
							"cpu_limit":   2,
							"gpu_request": 0,
							"gpu_limit":   0,
							"storage": map[string]interface{}{
								"data": map[string]interface{}{
									"size":          "500Gi",
									"accessModes":   []string{"ReadWriteOnce"},
									"disk_replicas": 5,
									"storageClass":  "silver",
								},
								"log": map[string]interface{}{
									"size":         "20Gi",
									"accessMode":   "ReadWriteOnce",
									"storageClass": "silver",
									"disk_replicas": 0,
								},
							},
							"memory_request": 1,
							"memory_limit":   4,
						},
						"image":    "172.16.1.99/gold/hdfs:transwarp-5.2",
						"priority": 10,
						"replicas": 3,
						"env_list": []interface{}{},
					},
					"hdfsjournalnode": map[string]interface{}{
						"replicas":         3,
						"env_list":         []interface{}{},
						"use_host_network": false,
						"resources": map[string]interface{}{
							"storage": map[string]interface{}{
								"data": map[string]interface{}{
									"storageClass":  "silver",
									"size":          "500Gi",
									"accessModes":   []string{"ReadWriteOnce"},
									"disk_replicas": 0,
								},
								"log": map[string]interface{}{
									"storageClass": "silver",
									"size":         "20Gi",
									"accessMode":   "ReadWriteOnce",
									"disk_replicas": 0,
								},
							},
							"memory_request": 1,
							"memory_limit":   4,
							"cpu_request":    0.5,
							"cpu_limit":      2,
							"gpu_request":    0,
							"gpu_limit":      0,
						},
						"image":    "172.16.1.99/gold/hdfs:transwarp-5.2",
						"priority": 10,
					},
					"httpfs": map[string]interface{}{
						"resources": map[string]interface{}{
							"gpu_limit": 0,
							"storage": map[string]interface{}{
								"log": map[string]interface{}{
									"accessMode":   "ReadWriteOnce",
									"storageClass": "silver",
									"size":         "20Gi",
									"disk_replicas": 0,
								},
							},
							"memory_request": 1,
							"memory_limit":   4,
							"cpu_request":    0.5,
							"cpu_limit":      2,
							"gpu_request":    0,
						},
						"image":            "172.16.1.99/gold/httpfs:transwarp-5.2",
						"priority":         10,
						"replicas":         2,
						"env_list":         []interface{}{},
						"use_host_network": false,
					},
				},
				"Advance_Config": map[string]interface{}{
					"hdfs":        map[string]interface{}{},
					"core_site":   map[string]interface{}{},
					"hdfs_site":   map[string]interface{}{},
					"httpfs_site": map[string]interface{}{},
				},
				"Transwarp_Config": map[string]interface{}{
					"security": map[string]interface{}{
						"auth_type":                      "none",
						"guardian_principal_host":        "tos",
						"guardian_principal_user":        "hdfs",
						"guardian_spnego_principal_host": "tos",
						"guardian_spnego_principal_user": "HTTP",
					},
					"Ingress":                         map[string]interface{}{},
					"Transwarp_Metric_Enable":         true,
					"Transwarp_Auto_Injected_Volumes": []interface{}{},
				},
			},
		},
	}

	for _, test := range tests {
		prettyChartParams := UserInputParams{}
		err := yaml.Unmarshal([]byte(test.prettyParamsStr), &prettyChartParams)
		if err != nil {
			klog.Error(err.Error())
		}
		assert.IsType(t, nil, err)

		request := &release.ReleaseRequest{
			ConfigValues:        test.configValues,
			ReleasePrettyParams: convertUserInputParams(&prettyChartParams),
		}
		processPrettyParams(request)

		unifiedResult, err := util.UnifyConfigValue(test.result)
		assert.IsType(t, nil, err)

		unifiedConfig, err := util.UnifyConfigValue(request.ConfigValues)
		assert.IsType(t, nil, err)

		assert.Equal(t, unifiedResult, unifiedConfig)
	}
}

func Test_mapKey(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		data  map[string]interface{}
	}{
		{
			key:   "Advance_Config.zookeeper[\"zookeeper.leader.elect.port\"].kkk",
			value: "2000",
			data: map[string]interface{}{
				"Advance_Config": map[string]interface{}{
					"zookeeper": map[string]interface{}{
						"zookeeper.leader.elect.port": map[string]interface{}{
							"kkk": "2000",
						},
					},
				},
			},
		},
		{
			key:   "Advance_Config.zookeeper[\"zookeeper.leader.elect.port\"]",
			value: "2000",
			data: map[string]interface{}{
				"Advance_Config": map[string]interface{}{
					"zookeeper": map[string]interface{}{
						"zookeeper.leader.elect.port": "2000",
					},
				},
			},
		},
		{
			key:   "Advance_Config.zookeeper",
			value: "2000",
			data: map[string]interface{}{
				"Advance_Config": map[string]interface{}{
					"zookeeper": "2000",
				},
			},
		},
	}

	for _, test := range tests {
		data := map[string]interface{}{}
		mapKey(test.key, test.value, data)
		assert.Equal(t, test.data, data)
	}
}

var hdfsPrettyParams = `
commonConfig:
 roles:
 - name: hdfsnamenode
   description: "hdfsnamenode服务"
   baseConfig:
   - variable: image
     default: 172.16.1.99/gold/hdfs:transwarp-5.2
     description: "镜像"
     type: string
   - variable: priority
     default: 10
     description: "优先级"
     type: number
   - variable: replicas
     default: 2
     description: "副本个数"
     type: number
   - variable: env_list
     default: []
     description: "额外环境变量"
     type: list
   - variable: use_host_network
     default: false
     description: "是否使用主机网络"
     type: bool
   resouceConfig:
     cpu_limit: 2
     cpu_request: 1
     memory_limit: 8
     memory_request: 4
     gpu_limit: 0
     gpu_request: 0
     extra_resources: []
     storage:
     - name: data
       type: pvc
       storageClass: "silver"
       size: "100Gi"
       accessModes: ["ReadWriteOnce"]
       limit: {}
     - name: log
       type: tosDisk
       storageClass: "silver"
       size: "20Gi"
       accessMode: "ReadWriteOnce"
       limit: {}
 - name: hdfszkfc
   description: "hdfszkfc服务"
   baseConfig:
   - variable: image
     default: 172.16.1.99/transwarp1111111111/hdfs:transwarp-5.2
     description: "镜像"
     type: string
   - variable: env_list
     default: []
     description: "额外环境变量"
     type: list
   resouceConfig:
     cpu_limit: 0.5
     cpu_request: 0.1
     memory_limit: 1
     memory_request: 0.5
     gpu_limit: 0
     gpu_request: 0
     extra_resources: []
 - name: hdfsdatanode
   description: "hdfsdatanode服务"
   baseConfig:
   - variable: image
     default: 172.16.1.99/gold/hdfs:transwarp-5.2
     description: "镜像"
     type: string
   - variable: priority
     default: 10
     description: "优先级"
     type: number
   - variable: replicas
     default: 3
     description: "副本个数"
     type: number
   - variable: env_list
     default: []
     description: "额外环境变量"
     type: list
   - variable: use_host_network
     default: false
     description: "是否使用主机网络"
     type: bool
   resouceConfig:
     cpu_limit: 2
     cpu_request: 0.5
     memory_limit: 4
     memory_request: 1
     gpu_limit: 0
     gpu_request: 0
     extra_resources: []
     storage:
     - name: data
       type: pvc
       storageClass: "silver"
       size: "500Gi"
       accessModes: ["ReadWriteOnce"]
       limit: {}
       disk_replicas: 5
     - name: log
       type: tosDisk
       storageClass: "silver"
       size: "20Gi"
       accessMode: "ReadWriteOnce"
       limit: {}
 - name: hdfsjournalnode
   description: "hdfsjournalnode服务"
   baseConfig:
   - variable: image
     default: 172.16.1.99/gold/hdfs:transwarp-5.2
     description: "镜像"
     type: string
   - variable: priority
     default: 10
     description: "优先级"
     type: number
   - variable: replicas
     default: 3
     description: "副本个数"
     type: number
   - variable: env_list
     default: []
     description: "额外环境变量"
     type: list
   - variable: use_host_network
     default: false
     description: "是否使用主机网络"
     type: bool
   resouceConfig:
     cpu_limit: 2
     cpu_request: 0.5
     memory_limit: 4
     memory_request: 1
     gpu_limit: 0
     gpu_request: 0
     extra_resources: []
     storage:
     - name: data
       type: pvc
       storageClass: "silver"
       size: "500Gi"
       accessModes: ["ReadWriteOnce"]
       limit: {}
     - name: log
       type: tosDisk
       storageClass: "silver"
       size: "20Gi"
       accessMode: "ReadWriteOnce"
       limit: {}
 - name: httpfs
   description: "httpfs服务"
   baseConfig:
   - variable: image
     default: 172.16.1.99/gold/httpfs:transwarp-5.2
     description: "镜像"
     type: string
   - variable: priority
     default: 10
     description: "优先级"
     type: number
   - variable: replicas
     default: 2
     description: "副本个数"
     type: number
   - variable: env_list
     default: []
     description: "额外环境变量"
     type: list
   - variable: use_host_network
     default: false
     description: "是否使用主机网络"
     type: bool
   resouceConfig:
     cpu_limit: 2
     cpu_request: 0.5
     memory_limit: 4
     memory_request: 1
     gpu_limit: 0
     gpu_request: 0
     extra_resources: []
     storage:
     - name: log
       type: tosDisk
       storageClass: "silver"
       size: "20Gi"
       accessMode: "ReadWriteOnce"
       limit: {}
transwarpBundleConfig:
- variable: Transwarp_Config.Transwarp_Metric_Enable
  default: true
  description: "是否开启组件metrics服务"
  type: bool
- variable: Transwarp_Config.Transwarp_Auto_Injected_Volumes
  default: []
  description: "自动挂载keytab目录"
  type: list
- variable: Transwarp_Config.security.auth_type
  default: "none"
  description: "开启安全类型"
  type: string
- variable: Transwarp_Config.security.guardian_principal_host
  default: "tos"
  description: "开启安全服务Principal主机名"
  type: string
- variable: Transwarp_Config.security.guardian_principal_user
  default: "hdfs"
  description: "开启安全服务Principal用户名"
  type: string
- variable: Transwarp_Config.security.guardian_spnego_principal_host
  default: "tos"
  description: "Httpfs开启安全服务Principal主机名"
  type: string
- variable: Transwarp_Config.security.guardian_spnego_principal_user
  default: "HTTP"
  description: "Httpfs开启安全服务Principal用户名"
  type: string
- variable: Transwarp_Config.Ingress
  default: {}
  description: "HDFS Ingress配置参数"
  type: yaml
advanceConfig:
- variable: Advance_Config.hdfs
  default: {}
  description: "hdfs guardian配置"
  type: yaml
- variable: Advance_Config.core_site
  default: {}
  description: "hdfs core-site配置"
  type: yaml
- variable: Advance_Config.hdfs_site
  default: {}
  description: "hdfs hdfs-site配置"
  type: yaml
- variable: Advance_Config.httpfs_site
  default: {}
  description: "hdfs httpfs-site配置"
  type: yaml
`
