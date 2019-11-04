package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"transwarp/release-config/pkg/apis/transwarp/v1beta1"
	"WarpCloud/walm/pkg/models/common"
)

func TestConvertReleaseConfigFromK8s(t *testing.T) {
	tests := []struct {
		oriReleaseConfig *v1beta1.ReleaseConfig
		releaseConfig    *k8s.ReleaseConfig
		err              error
	}{
		{
			oriReleaseConfig: &v1beta1.ReleaseConfig{
				TypeMeta: metav1.TypeMeta{
					Kind: "ReleaseConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-releaseConfig",
					Namespace: "test-namespace",
					Labels:    map[string]string{"test1": "test1"},
				},
				Spec: v1beta1.ReleaseConfigSpec{
					ChartName:       "zookeeper",
					ChartAppVersion: "6.1",
					ChartVersion:    "6.1.0",
					Repo:            "qa",
					IsomateConfig: &v1beta1.IsomateConfig{
						DefaultIsomateName: "x86",
						Isomates: []*v1beta1.Isomate{
							{
								Name: "x86",
								Plugins: []*v1beta1.ReleasePlugin{
									{
										Name: "test",
									},
								},
								ConfigValues: map[string]interface{}{
									"test": "test",
								},
							},
						},
					},
					ChartWalmVersion: "v1",
				},
			},
			releaseConfig: &k8s.ReleaseConfig{
				Meta: k8s.Meta{
					Name:      "test-releaseConfig",
					Namespace: "test-namespace",
					Kind:      "ReleaseConfig",
					State: k8s.State{
						Status:  "",
						Reason:  "",
						Message: "",
					},
				},
				Labels:          map[string]string{"test1": "test1"},
				ChartName:       "zookeeper",
				ChartVersion:    "6.1.0",
				ChartAppVersion: "6.1",
				Repo:            "qa",
				IsomateConfig: &k8s.IsomateConfig{
					DefaultIsomateName: "x86",
					Isomates: []*k8s.Isomate{
						{
							Name: "x86",
							Plugins: []*k8s.ReleasePlugin{
								{
									Name: "test",
								},
							},
							ConfigValues: map[string]interface{}{
								"test": "test",
							},
						},
					},
				},
				ChartWalmVersion: common.WalmVersionV1,
			},
		},
		{
			oriReleaseConfig: nil,
			releaseConfig:    nil,
			err:              nil,
		},
	}

	for _, test := range tests {
		releaseConfig, err := ConvertReleaseConfigFromK8s(test.oriReleaseConfig)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.releaseConfig, releaseConfig)
	}
}

func Test_ConvertIsomateConfigToK8s(t *testing.T) {
	tests := []struct {
		oriIsomateConfig *k8s.IsomateConfig
		isomateConfig    *v1beta1.IsomateConfig
	}{
		{
			oriIsomateConfig: &k8s.IsomateConfig{
				DefaultIsomateName: "x86",
				Isomates: []*k8s.Isomate{
					{
						Name: "x86",
						Plugins: []*k8s.ReleasePlugin{
							{
								Name: "test",
							},
						},
						ConfigValues: map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
			isomateConfig: &v1beta1.IsomateConfig{
				DefaultIsomateName: "x86",
				Isomates: []*v1beta1.Isomate{
					{
						Name: "x86",
						Plugins: []*v1beta1.ReleasePlugin{
							{
								Name: "test",
							},
						},
						ConfigValues: map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		isomateConfig := ConvertIsomateConfigToK8s(test.oriIsomateConfig)
		assert.Equal(t, test.isomateConfig, isomateConfig)
	}
}
