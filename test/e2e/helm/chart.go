package helm

import (
	. "github.com/onsi/gomega"
	. "github.com/onsi/ginkgo"
	"WarpCloud/walm/pkg/helm/impl"
	"WarpCloud/walm/test/e2e/framework"
	"WarpCloud/walm/pkg/k8s/cache/informer"
	"WarpCloud/walm/pkg/setting"
	"WarpCloud/walm/pkg/models/release"
	"path/filepath"
	"WarpCloud/walm/pkg/models/common"
	"encoding/json"
)

var _ = Describe("HelmChart", func() {

	var (
		helm     *impl.Helm
		stopChan chan struct{}
	)

	BeforeEach(func() {
		stopChan = make(chan struct{})
		k8sCache := informer.NewInformer(framework.GetK8sClient(), framework.GetK8sReleaseConfigClient(), 0, stopChan)
		registryClient, err := impl.NewRegistryClient(setting.Config.ChartImageConfig)
		Expect(err).NotTo(HaveOccurred())

		helm, err = impl.NewHelm(setting.Config.RepoList, registryClient, k8sCache, framework.GetKubeClient())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		close(stopChan)
	})

	It("test get chart auto dependency", func() {
		currentFilePath, err := framework.GetCurrentFilePath()
		Expect(err).NotTo(HaveOccurred())

		chartPath := filepath.Join(filepath.Dir(currentFilePath), "../../resources/helm/kafka-6.1.0.tgz")
		Expect(err).NotTo(HaveOccurred())

		testRepoUrl := ""
		for _, repo := range helm.GetRepoList().Items {
			if repo.TenantRepoName == framework.TestChartRepoName {
				testRepoUrl = repo.TenantRepoURL
			}
		}
		Expect(testRepoUrl).NotTo(Equal(""))

		err = framework.PushChartToRepo(testRepoUrl, chartPath)
		Expect(err).NotTo(HaveOccurred())

		By("test v2 chart")
		dependencies, err := helm.GetChartAutoDependencies(framework.TestChartRepoName, "kafka", "6.1.0")
		Expect(dependencies).To(Equal([]string{"zookeeper"}))

		By("test v1 chart")
		dependencies, err = helm.GetChartAutoDependencies(framework.TestChartRepoName, framework.V1ZookeeperChartName, framework.V1ZookeeperChartVersion)
		Expect(dependencies).To(Equal([]string{"guardian"}))
	})

	It("test get chart list", func() {
		chartList, err := helm.GetChartList(framework.TestChartRepoName)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(chartList.Items) >= 1).To(BeTrue())

		getChart := func(chartName, chartVersion string) *release.ChartInfo {
			for _, chartInfo := range chartList.Items {
				if chartInfo.ChartName == chartName && chartInfo.ChartVersion == chartVersion {
					return chartInfo
				}
			}
			return nil
		}

		testChart := getChart(framework.TomcatChartName, framework.TomcatChartVersion)
		Expect(testChart).NotTo(BeNil())
		Expect(testChart.ChartAppVersion).To(Equal("7"))
		Expect(testChart.ChartDescription).To(Equal("开源的轻量级Web应用服务器，支持HTML、JS等静态资源的处理，可以作为轻量级Web服务器使用。"))
	})

	It("test get chart", func() {
		chartInfo, err := helm.GetChartDetailInfo(framework.TestChartRepoName, framework.TomcatChartName, framework.TomcatChartVersion)
		Expect(err).NotTo(HaveOccurred())
		Expect(chartInfo.Icon).NotTo(Equal(""))
		Expect(chartInfo.Advantage).NotTo(Equal(""))
		Expect(chartInfo.Architecture).NotTo(Equal(""))
		Expect(chartInfo.ChartInfo).To(Equal(release.ChartInfo{
			ChartVersion:     framework.TomcatChartVersion,
			ChartName:        framework.TomcatChartName,
			ChartAppVersion:  "7",
			DefaultValue:     "{\"affinity\":{},\"app\":{\"path\":\"/sample\"},\"deploy\":{\"directory\":\"/usr/local/tomcat/webapps\"},\"image\":{\"pullPolicy\":\"Always\",\"pullSecrets\":[],\"tomcat\":{\"image\":\"tomcat:7.0\"},\"webarchive\":{\"image\":\"ananwaresystems/webarchive:1.0\"}},\"ingress\":{\"annotations\":{},\"enabled\":false,\"hosts\":[\"chart-example.local\"],\"path\":\"/\",\"tls\":[]},\"livenessProbe\":{\"initialDelaySeconds\":60,\"periodSeconds\":30},\"nodeSelector\":{},\"readinessProbe\":{\"failureThreshold\":6,\"initialDelaySeconds\":60,\"periodSeconds\":30},\"replicaCount\":1,\"resources\":{\"limits\":{\"cpu\":0.2,\"memory\":\"200Mi\"},\"requests\":{\"cpu\":0.1,\"memory\":\"100Mi\"}},\"service\":{\"externalPort\":80,\"internalPort\":8080,\"name\":\"http\",\"type\":\"NodePort\"},\"tolerations\":[]}",
			ChartDescription: "开源的轻量级Web应用服务器，支持HTML、JS等静态资源的处理，可以作为轻量级Web服务器使用。",
			ChartEngine:      "transwarp",
			MetaInfo: &release.ChartMetaInfo{
				FriendlyName: "tomcat",
				Categories:   []string{"tdh", "tdc"},
				ChartRoles: []*release.MetaRoleConfig{
					{
						Name:        "webarchive",
						Description: "webarchive",
						Type:        "initContainer",
						RoleBaseConfig: &release.MetaRoleBaseConfig{
							Image: &release.MetaStringConfig{
								Type:         "string",
								MapKey:       "image.webarchive.image",
								Description:  "镜像",
								DefaultValue: "ananwaresystems/webarchive:1.0",
								Required:     true,
							},
						},
						RoleHealthCheckConfig: &release.MetaHealthCheckConfig{
							LivenessProbe:  &release.MetaHealthProbeConfig{},
							ReadinessProbe: &release.MetaHealthProbeConfig{},
						},
					},
					{
						Name:        "tomcat",
						Description: "tomcat",
						Type:        "container",
						RoleBaseConfig: &release.MetaRoleBaseConfig{
							Image: &release.MetaStringConfig{
								Type:         "string",
								MapKey:       "image.tomcat.image",
								Description:  "镜像",
								DefaultValue: "tomcat:7.0",
								Required:     true,
							},
							Replicas: &release.MetaIntConfig{
								IntConfig: release.IntConfig{
									Type:         "number",
									MapKey:       "replicaCount",
									Description:  "副本个数",
									DefaultValue: 1,
									Required:     true,
								},
							},
							Others: []*release.MetaCommonConfig{
								{
									Name:         "path",
									MapKey:       "app.path",
									Description:  "appPath",
									Type:         "string",
									Required:     true,
									DefaultValue: "\"/sample\"",
								},
							},
						},
						RoleResourceConfig: &release.MetaResourceConfig{
							LimitsMemory: &release.MetaResourceMemoryConfig{
								IntConfig: release.IntConfig{
									MapKey:       "resources.limits.memory",
									DefaultValue: 200,
								},
							},
							RequestsMemory: &release.MetaResourceMemoryConfig{
								IntConfig: release.IntConfig{
									MapKey:       "resources.requests.memory",
									DefaultValue: 100,
								},
							},
							LimitsCpu: &release.MetaResourceCpuConfig{
								FloatConfig: release.FloatConfig{
									MapKey:       "resources.limits.cpu",
									DefaultValue: 0.2,
								},
							},
							RequestsCpu: &release.MetaResourceCpuConfig{
								FloatConfig: release.FloatConfig{
									MapKey:       "resources.requests.cpu",
									DefaultValue: 0.1,
								},
							},
						},
						RoleHealthCheckConfig: &release.MetaHealthCheckConfig{
							LivenessProbe:  &release.MetaHealthProbeConfig{},
							ReadinessProbe: &release.MetaHealthProbeConfig{},
						},
					},
				},
			},
			WalmVersion: common.WalmVersionV2,
		}))
	})

	It("test get v1 chart", func() {
		chartInfo, err := helm.GetChartDetailInfo(framework.TestChartRepoName, framework.V1ZookeeperChartName, framework.V1ZookeeperChartVersion)
		Expect(err).NotTo(HaveOccurred())
		Expect(chartInfo.Icon).To(Equal(""))
		Expect(chartInfo.Advantage).To(Equal(""))
		Expect(chartInfo.Architecture).To(Equal(""))
		chartInfoString, err := json.Marshal(chartInfo.ChartInfo)
		Expect(err).NotTo(HaveOccurred())
		expectedChartInfo := release.ChartInfo{
			ChartVersion:     framework.V1ZookeeperChartVersion,
			ChartName:        framework.V1ZookeeperChartName,
			ChartAppVersion:  "5.2",
			DefaultValue:     "{\"Customized_Instance_Selector\":{},\"Customized_Namespace\":\"\",\"Transwarp_Install_ID\":\"\",\"Transwarp_Install_Namespace\":\"\",\"Transwarp_License_Address\":\"\"}",
			ChartDescription: "分布式配置服务",
			ChartEngine:      "transwarp",
			DependencyCharts: []release.ChartDependencyInfo{
				{
					ChartName: "guardian",
					MinVersion: 6,
					MaxVersion: 5.2,
					DependencyOptional: true,
				},
			},
			ChartPrettyParams: &release.PrettyChartParams{
				CommonConfig: release.CommonConfig{
					Roles: []*release.RoleConfig{
						{
							Name: "zookeeper",
							Description: "zookeeper服务",
							RoleBaseConfig: []*release.BaseConfig{
								{
									ValueName: "image",
									DefaultValue: "zookeeper:transwarp-5.2",
									ValueDescription: "镜像",
									ValueType: "string",
								},
								{
									ValueName: "replicas",
									DefaultValue: 3,
									ValueDescription: "副本个数",
									ValueType: "number",
								},
								{
									ValueName: "env_list",
									DefaultValue: []interface{}{},
									ValueDescription: "额外环境变量",
									ValueType: "list",
								},
								{
									ValueName: "use_host_network",
									DefaultValue: false,
									ValueDescription: "是否使用主机网络",
									ValueType: "bool",
								},
								{
									ValueName: "priority",
									DefaultValue: 0,
									ValueDescription: "优先级",
									ValueType: "number",
								},
							},
							RoleResourceConfig: &release.ResourceConfig{
								CpuLimit: 2,
								CpuRequest: 0.1,
								MemoryLimit: 4,
								MemoryRequest: 1,
								ResourceStorageList: []release.ResourceStorageConfig{
									{
										Name: "data",
										StorageClass: "silver",
										Size: "30Gi",
										AccessModes: []string{"ReadWriteOnce"},
										StorageType: "pvc",
									},
								},
							},
						},
					},
				},
				TranswarpBaseConfig: []*release.BaseConfig{
					{
						ValueName: "Transwarp_Config.Transwarp_Auto_Injected_Volumes",
						DefaultValue: []interface{}{},
						ValueDescription: "自动挂载keytab目录",
						ValueType: "list",
					},
					{
						ValueName: "Transwarp_Config.security.auth_type",
						DefaultValue: "none",
						ValueDescription: "开启安全类型",
						ValueType: "string",
					},
					{
						ValueName: "Transwarp_Config.security.guardian_principal_host",
						DefaultValue: "tos",
						ValueDescription: "开启安全服务Principal主机名",
						ValueType: "string",
					},
					{
						ValueName: "Transwarp_Config.security.guardian_principal_user",
						DefaultValue: "zookeeper",
						ValueDescription: "开启安全服务Principal用户名",
						ValueType: "string",
					},
				},
				AdvanceConfig: []*release.BaseConfig{
					{
						ValueName: "Advance_Config.zookeeper[\"zookeeper.client.port\"]",
						DefaultValue: 2181,
						ValueDescription: "zookeeper client监听端口",
						ValueType: "number",
					},
					{
						ValueName: "Advance_Config.zookeeper[\"zookeeper.peer.communicate.port\"]",
						DefaultValue: 2888,
						ValueDescription: "zookeeper peer communicate端口",
						ValueType: "number",
					},
					{
						ValueName: "Advance_Config.zookeeper[\"zookeeper.leader.elect.port\"]",
						DefaultValue: 3888,
						ValueDescription: "zookeeper leader elect端口",
						ValueType: "number",
					},
					{
						ValueName: "Advance_Config.zookeeper[\"zookeeper.jmxremote.port\"]",
						DefaultValue: 9911,
						ValueDescription: "zookeeper jmx端口",
						ValueType: "number",
					},
					{
						ValueName: "Advance_Config.zoo_cfg",
						DefaultValue: map[string]interface{}{},
						ValueDescription: "zookeeper zoo.cfg配置",
						ValueType: "yaml",
					},
				},
			},
			WalmVersion: common.WalmVersionV1,
		}
		expectedChartInfoString, err := json.Marshal(expectedChartInfo)
		Expect(err).NotTo(HaveOccurred())
		Expect(chartInfoString).To(Equal(expectedChartInfoString))
	})

	It("test get chart by image", func() {
		chartInfo, err := helm.GetDetailChartInfoByImage(framework.GetTomcatChartImage())
		Expect(err).NotTo(HaveOccurred())
		Expect(chartInfo.Icon).NotTo(Equal(""))
		Expect(chartInfo.Advantage).NotTo(Equal(""))
		Expect(chartInfo.Architecture).NotTo(Equal(""))
		Expect(chartInfo.ChartInfo).To(Equal(release.ChartInfo{
			ChartVersion:     framework.TomcatChartVersion,
			ChartName:        framework.TomcatChartName,
			ChartAppVersion:  "7",
			DefaultValue:     "{\"affinity\":{},\"app\":{\"path\":\"/sample\"},\"deploy\":{\"directory\":\"/usr/local/tomcat/webapps\"},\"image\":{\"pullPolicy\":\"Always\",\"pullSecrets\":[],\"tomcat\":{\"image\":\"tomcat:7.0\"},\"webarchive\":{\"image\":\"ananwaresystems/webarchive:1.0\"}},\"ingress\":{\"annotations\":{},\"enabled\":false,\"hosts\":[\"chart-example.local\"],\"path\":\"/\",\"tls\":[]},\"livenessProbe\":{\"initialDelaySeconds\":60,\"periodSeconds\":30},\"nodeSelector\":{},\"readinessProbe\":{\"failureThreshold\":6,\"initialDelaySeconds\":60,\"periodSeconds\":30},\"replicaCount\":1,\"resources\":{\"limits\":{\"cpu\":0.2,\"memory\":\"200Mi\"},\"requests\":{\"cpu\":0.1,\"memory\":\"100Mi\"}},\"service\":{\"externalPort\":80,\"internalPort\":8080,\"name\":\"http\",\"type\":\"NodePort\"},\"tolerations\":[]}",
			ChartDescription: "开源的轻量级Web应用服务器，支持HTML、JS等静态资源的处理，可以作为轻量级Web服务器使用。",
			ChartEngine:      "transwarp",
			MetaInfo: &release.ChartMetaInfo{
				FriendlyName: "tomcat",
				Categories:   []string{"tdh", "tdc"},
				ChartRoles: []*release.MetaRoleConfig{
					{
						Name:        "webarchive",
						Description: "webarchive",
						Type:        "initContainer",
						RoleBaseConfig: &release.MetaRoleBaseConfig{
							Image: &release.MetaStringConfig{
								Type:         "string",
								MapKey:       "image.webarchive.image",
								Description:  "镜像",
								DefaultValue: "ananwaresystems/webarchive:1.0",
								Required:     true,
							},
						},
						RoleHealthCheckConfig: &release.MetaHealthCheckConfig{
							LivenessProbe:  &release.MetaHealthProbeConfig{},
							ReadinessProbe: &release.MetaHealthProbeConfig{},
						},
					},
					{
						Name:        "tomcat",
						Description: "tomcat",
						Type:        "container",
						RoleBaseConfig: &release.MetaRoleBaseConfig{
							Image: &release.MetaStringConfig{
								Type:         "string",
								MapKey:       "image.tomcat.image",
								Description:  "镜像",
								DefaultValue: "tomcat:7.0",
								Required:     true,
							},
							Replicas: &release.MetaIntConfig{
								IntConfig: release.IntConfig{
									Type:         "number",
									MapKey:       "replicaCount",
									Description:  "副本个数",
									DefaultValue: 1,
									Required:     true,
								},
							},
							Others: []*release.MetaCommonConfig{
								{
									Name:         "path",
									MapKey:       "app.path",
									Description:  "appPath",
									Type:         "string",
									Required:     true,
									DefaultValue: "\"/sample\"",
								},
							},
						},
						RoleResourceConfig: &release.MetaResourceConfig{
							LimitsMemory: &release.MetaResourceMemoryConfig{
								IntConfig: release.IntConfig{
									MapKey:       "resources.limits.memory",
									DefaultValue: 200,
								},
							},
							RequestsMemory: &release.MetaResourceMemoryConfig{
								IntConfig: release.IntConfig{
									MapKey:       "resources.requests.memory",
									DefaultValue: 100,
								},
							},
							LimitsCpu: &release.MetaResourceCpuConfig{
								FloatConfig: release.FloatConfig{
									MapKey:       "resources.limits.cpu",
									DefaultValue: 0.2,
								},
							},
							RequestsCpu: &release.MetaResourceCpuConfig{
								FloatConfig: release.FloatConfig{
									MapKey:       "resources.requests.cpu",
									DefaultValue: 0.1,
								},
							},
						},
						RoleHealthCheckConfig: &release.MetaHealthCheckConfig{
							LivenessProbe:  &release.MetaHealthProbeConfig{},
							ReadinessProbe: &release.MetaHealthProbeConfig{},
						},
					},
				},
			},
			WalmVersion: common.WalmVersionV2,
		}))
	})
})
