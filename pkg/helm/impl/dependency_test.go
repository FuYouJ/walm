package impl

import (
	"testing"
	"WarpCloud/walm/pkg/models/common"
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/stretchr/testify/assert"
	k8sMocks "WarpCloud/walm/pkg/k8s/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/release"
)

func Test_buildCompatibleChartWalmVersion(t *testing.T) {
	tests := []struct {
		releaseConfig    *k8s.ReleaseConfig
		chartWalmVersion common.WalmVersion
	}{
		{
			releaseConfig: &k8s.ReleaseConfig{
				ChartWalmVersion: common.WalmVersionV1,
			},
			chartWalmVersion: common.WalmVersionV1,
		},
		{
			releaseConfig: &k8s.ReleaseConfig{
			},
			chartWalmVersion: "",
		},
		{
			releaseConfig: &k8s.ReleaseConfig{
				OutputConfig: map[string]interface{}{
					"test": "test",
				},
			},
			chartWalmVersion: common.WalmVersionV2,
		},
		{
			releaseConfig: &k8s.ReleaseConfig{
				OutputConfig: map[string]interface{}{
					"provides": map[string]interface{}{
						"testKey": map[string]interface{}{
							"immediate_value": map[string]interface{}{
								"test": "test",
							},
						},
					},
				},
			},
			chartWalmVersion: common.WalmVersionV1,
		},
	}

	for _, test := range tests {
		version := buildCompatibleChartWalmVersion(test.releaseConfig)
		assert.Equal(t, test.chartWalmVersion, version)
	}
}

func TestHelm_getOutputConfigValuesForChartV2(t *testing.T) {
	var mockK8sCache *k8sMocks.Cache
	mockHelm := &Helm{}

	refreshMocks := func() {
		mockK8sCache = &k8sMocks.Cache{}
		mockHelm.k8sCache = mockK8sCache
	}
	tests := []struct {
		initMock                   func()
		dependencyChartWalmVersion common.WalmVersion
		outputConfig               map[string]interface{}
		err                        error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", k8s.InstanceKind, "testns", "testnm").Return(&k8s.ApplicationInstance{
					DependencyMeta: &k8s.DependencyMeta{
						Provides: map[string]k8s.DependencyProvide{
							"testKey": {
								ImmediateValue: map[string]interface{}{
									"test": "test",
								},
							},
						},
					},
				}, nil)
			},
			dependencyChartWalmVersion: common.WalmVersionV1,
			outputConfig: map[string]interface{}{
				"provides": map[string]interface{}{
					"testKey": map[string]interface{}{
						"immediate_value": map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", k8s.InstanceKind, "testns", "testnm").Return(nil, errors.New(""))
			},
			err: errors.New(""),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(func(kind k8s.ResourceKind, namespace, name string) k8s.Resource {
					if kind == k8s.InstanceKind {
						return nil
					} else if kind == k8s.ReleaseConfigKind {
						return &k8s.ReleaseConfig{
							OutputConfig: map[string]interface{}{
								"test": "test",
							},
							ChartWalmVersion: common.WalmVersionV2,
						}
					} else {
						return nil
					}
				}, func(kind k8s.ResourceKind, namespace, name string) error {
					if kind == k8s.InstanceKind {
						return errorModel.NotFoundError{}
					} else {
						return nil
					}
				})
			},
			dependencyChartWalmVersion: common.WalmVersionV2,
			outputConfig: map[string]interface{}{
				"test": "test",
			},
		},
	}

	for _, test := range tests {
		test.initMock()
		version, outputConfig, err := mockHelm.getOutputConfigValuesForChartV2("testns", "testnm")
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.outputConfig, outputConfig)
		assert.Equal(t, test.dependencyChartWalmVersion, version)

		mockK8sCache.AssertExpectations(t)
	}
}

func TestHelm_getDependencyOutputConfigsForChartV2(t *testing.T) {
	var mockK8sCache *k8sMocks.Cache
	mockHelm := &Helm{}

	refreshMocks := func() {
		mockK8sCache = &k8sMocks.Cache{}
		mockHelm.k8sCache = mockK8sCache
	}
	tests := []struct {
		initMock             func()
		dependencies         map[string]string
		chartInfo            *release.ChartDetailInfo
		strict               bool
		dependencyConfigs    map[string]interface{}
		modifiedDependencies map[string]string
		err                  error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", k8s.InstanceKind, "testns", "testnm").Return(&k8s.ApplicationInstance{
					DependencyMeta: &k8s.DependencyMeta{
						Provides: map[string]k8s.DependencyProvide{
							"testKey": {
								ImmediateValue: map[string]interface{}{
									"test": "test",
								},
							},
						},
					},
				}, nil)
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					MetaInfo: &release.ChartMetaInfo{
						ChartDependenciesInfo: []*release.ChartDependencyMetaInfo{
							{
								Name:           "testDependencyChart",
								AliasConfigVar: "testKey",
							},
						},
					},
				},
			},
			dependencies: map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs: map[string]interface{}{
				"testKey": map[string]interface{}{
					"test": "test",
				},
			},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(func(kind k8s.ResourceKind, namespace, name string) k8s.Resource {
					if kind == k8s.InstanceKind {
						return nil
					} else if kind == k8s.ReleaseConfigKind {
						return &k8s.ReleaseConfig{
							OutputConfig: map[string]interface{}{
								"test": "test",
							},
							ChartWalmVersion: common.WalmVersionV2,
						}
					} else {
						return nil
					}
				}, func(kind k8s.ResourceKind, namespace, name string) error {
					if kind == k8s.InstanceKind {
						return errorModel.NotFoundError{}
					} else {
						return nil
					}
				})
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					MetaInfo: &release.ChartMetaInfo{
						ChartDependenciesInfo: []*release.ChartDependencyMetaInfo{
							{
								Name:           "testDependencyChart",
								AliasConfigVar: "testKey",
							},
						},
					},
				},
			},
			dependencies: map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs: map[string]interface{}{
				"testKey": map[string]interface{}{
					"test": "test",
				},
			},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
		},
		{
			initMock: func() {
				refreshMocks()
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					MetaInfo: &release.ChartMetaInfo{
						ChartDependenciesInfo: []*release.ChartDependencyMetaInfo{
							{
								Name:           "testDependencyChart1",
								AliasConfigVar: "testKey",
							},
						},
					},
				},
			},
			dependencies: map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs: map[string]interface{}{
			},
			modifiedDependencies: map[string]string{},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(nil, errorModel.NotFoundError{})
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					MetaInfo: &release.ChartMetaInfo{
						ChartDependenciesInfo: []*release.ChartDependencyMetaInfo{
							{
								Name:           "testDependencyChart",
								AliasConfigVar: "testKey",
							},
						},
					},
				},
			},
			strict:               false,
			dependencies:         map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs:    map[string]interface{}{},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(nil, errorModel.NotFoundError{})
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					MetaInfo: &release.ChartMetaInfo{
						ChartDependenciesInfo: []*release.ChartDependencyMetaInfo{
							{
								Name:           "testDependencyChart",
								AliasConfigVar: "testKey",
							},
						},
					},
				},
			},
			strict:               true,
			dependencies:         map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs:    map[string]interface{}{},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
			err:                  errors.WithStack(errors.New("")),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(nil, errors.New(""))
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					MetaInfo: &release.ChartMetaInfo{
						ChartDependenciesInfo: []*release.ChartDependencyMetaInfo{
							{
								Name:           "testDependencyChart",
								AliasConfigVar: "testKey",
							},
						},
					},
				},
			},
			strict:               false,
			dependencies:         map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs:    map[string]interface{}{},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
			err:                  errors.New(""),
		},
	}

	for _, test := range tests {
		test.initMock()
		dependencyConfigs, err := mockHelm.getDependencyOutputConfigsForChartV2("testns", test.dependencies, test.chartInfo, test.strict)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.dependencyConfigs, dependencyConfigs)
		assert.Equal(t, test.modifiedDependencies, test.dependencies)

		mockK8sCache.AssertExpectations(t)
	}
}

func TestHelm_getDependencyMetaForChartV1(t *testing.T) {
	var mockK8sCache *k8sMocks.Cache
	mockHelm := &Helm{}

	refreshMocks := func() {
		mockK8sCache = &k8sMocks.Cache{}
		mockHelm.k8sCache = mockK8sCache
	}
	tests := []struct {
		initMock                   func()
		dependencyChartWalmVersion common.WalmVersion
		dependencyMeta             *k8s.DependencyMeta
		err                        error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", k8s.InstanceKind, "testns", "testnm").Return(&k8s.ApplicationInstance{
					DependencyMeta: &k8s.DependencyMeta{
						Provides: map[string]k8s.DependencyProvide{
							"testKey": {
								ImmediateValue: map[string]interface{}{
									"test": "test",
								},
							},
						},
					},
				}, nil)
			},
			dependencyChartWalmVersion: common.WalmVersionV1,
			dependencyMeta: &k8s.DependencyMeta{
				Provides: map[string]k8s.DependencyProvide{
					"testKey": {
						ImmediateValue: map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", k8s.InstanceKind, "testns", "testnm").Return(nil, errors.New(""))
			},
			err: errors.New(""),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(func(kind k8s.ResourceKind, namespace, name string) k8s.Resource {
					if kind == k8s.InstanceKind {
						return nil
					} else if kind == k8s.ReleaseConfigKind {
						return &k8s.ReleaseConfig{
							OutputConfig: map[string]interface{}{
								"test": "test",
							},
							ChartWalmVersion: common.WalmVersionV2,
						}
					} else {
						return nil
					}
				}, func(kind k8s.ResourceKind, namespace, name string) error {
					if kind == k8s.InstanceKind {
						return errorModel.NotFoundError{}
					} else {
						return nil
					}
				})
			},
			dependencyChartWalmVersion: common.WalmVersionV2,
			dependencyMeta: &k8s.DependencyMeta{
				Provides: map[string]k8s.DependencyProvide{
					dummyDependencyProvideKey: {
						ImmediateValue: map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(func(kind k8s.ResourceKind, namespace, name string) k8s.Resource {
					if kind == k8s.InstanceKind {
						return nil
					} else if kind == k8s.ReleaseConfigKind {
						return &k8s.ReleaseConfig{
							OutputConfig: map[string]interface{}{
								"provides": map[string]interface{}{
									"testKey": map[string]interface{}{
										"immediate_value": map[string]interface{}{
											"test": "test",
										},
									},
								},
							},
							ChartWalmVersion: common.WalmVersionV1,
						}
					} else {
						return nil
					}
				}, func(kind k8s.ResourceKind, namespace, name string) error {
					if kind == k8s.InstanceKind {
						return errorModel.NotFoundError{}
					} else {
						return nil
					}
				})
			},
			dependencyChartWalmVersion: common.WalmVersionV1,
			dependencyMeta: &k8s.DependencyMeta{
				Provides: map[string]k8s.DependencyProvide{
					"testKey": {
						ImmediateValue: map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test.initMock()
		version, dependencyMeta, err := mockHelm.getDependencyMetaForChartV1("testns", "testnm")
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.dependencyMeta, dependencyMeta)
		assert.Equal(t, test.dependencyChartWalmVersion, version)

		mockK8sCache.AssertExpectations(t)
	}
}

func TestHelm_getDependencyOutputConfigsForChartV1(t *testing.T) {
	var mockK8sCache *k8sMocks.Cache
	mockHelm := &Helm{}

	refreshMocks := func() {
		mockK8sCache = &k8sMocks.Cache{}
		mockHelm.k8sCache = mockK8sCache
	}
	tests := []struct {
		initMock             func()
		dependencies         map[string]string
		chartInfo            *release.ChartDetailInfo
		strict               bool
		dependencyConfigs    map[string]interface{}
		modifiedDependencies map[string]string
		err                  error
	}{
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", k8s.InstanceKind, "testns", "testnm").Return(&k8s.ApplicationInstance{
					DependencyMeta: &k8s.DependencyMeta{
						Provides: map[string]k8s.DependencyProvide{
							"testKey": {
								ImmediateValue: map[string]interface{}{
									"test": "test",
								},
							},
						},
					},
				}, nil)
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					DependencyCharts: []release.ChartDependencyInfo{
						{
							ChartName: "testDependencyChart",
							Requires: map[string]string{
								"testKey": "$(testKey)",
							},
						},
					},
				},
			},
			dependencies: map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs: map[string]interface{}{
				"testKey": map[string]interface{}{
					"test": "test",
				},
			},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(func(kind k8s.ResourceKind, namespace, name string) k8s.Resource {
					if kind == k8s.InstanceKind {
						return nil
					} else if kind == k8s.ReleaseConfigKind {
						return &k8s.ReleaseConfig{
							OutputConfig: map[string]interface{}{
								"test": "test",
							},
							ChartWalmVersion: common.WalmVersionV2,
						}
					} else {
						return nil
					}
				}, func(kind k8s.ResourceKind, namespace, name string) error {
					if kind == k8s.InstanceKind {
						return errorModel.NotFoundError{}
					} else {
						return nil
					}
				})
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					DependencyCharts: []release.ChartDependencyInfo{
						{
							ChartName: "testDependencyChart",
							Requires: map[string]string{
								"testKey": "$(testKey)",
							},
						},
					},
				},
			},
			dependencies: map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs: map[string]interface{}{
				"testKey": map[string]interface{}{
					"test": "test",
				},
			},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
		},
		{
			initMock: func() {
				refreshMocks()
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					DependencyCharts: []release.ChartDependencyInfo{
						{
							ChartName: "testDependencyChart1",
							Requires: map[string]string{
								"testKey": "$(testKey)",
							},
						},
					},
				},
			},
			dependencies: map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs: map[string]interface{}{
			},
			modifiedDependencies: map[string]string{},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(nil, errorModel.NotFoundError{})
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					DependencyCharts: []release.ChartDependencyInfo{
						{
							ChartName: "testDependencyChart",
							Requires: map[string]string{
								"testKey": "$(testKey)",
							},
						},
					},
				},
			},
			strict:               false,
			dependencies:         map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs:    map[string]interface{}{},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(nil, errorModel.NotFoundError{})
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					DependencyCharts: []release.ChartDependencyInfo{
						{
							ChartName: "testDependencyChart",
							Requires: map[string]string{
								"testKey": "$(testKey)",
							},
						},
					},
				},
			},
			strict:               true,
			dependencies:         map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs:    map[string]interface{}{},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
			err:                  errors.WithStack(errors.New("")),
		},
		{
			initMock: func() {
				refreshMocks()
				mockK8sCache.On("GetResource", mock.Anything, "testns", "testnm").Return(nil, errors.New(""))
			},
			chartInfo: &release.ChartDetailInfo{
				ChartInfo: release.ChartInfo{
					DependencyCharts: []release.ChartDependencyInfo{
						{
							ChartName: "testDependencyChart",
							Requires: map[string]string{
								"testKey": "$(testKey)",
							},
						},
					},
				},
			},
			strict:               false,
			dependencies:         map[string]string{"testDependencyChart": "testnm"},
			dependencyConfigs:    map[string]interface{}{},
			modifiedDependencies: map[string]string{"testDependencyChart": "testnm"},
			err:                  errors.New(""),
		},
	}

	for _, test := range tests {
		test.initMock()
		dependencyConfigs, err := mockHelm.getDependencyOutputConfigsForChartV1("testns", test.dependencies, test.chartInfo, test.strict)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.dependencyConfigs, dependencyConfigs)
		assert.Equal(t, test.modifiedDependencies, test.dependencies)

		mockK8sCache.AssertExpectations(t)
	}
}