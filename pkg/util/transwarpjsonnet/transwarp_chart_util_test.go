package transwarpjsonnet

import (
	"WarpCloud/walm/pkg/models/common"
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_buildAutoGenReleaseConfig(t *testing.T) {
	tests := []struct {
		releaseNamespace  string
		releaseName       string
		repo              string
		chartName         string
		chartVersion      string
		chartAppVersion   string
		labels            map[string]string
		dependencies      map[string]string
		dependencyConfigs map[string]interface{}
		userConfigs       map[string]interface{}
		chartImage        string
		isomateConfig     *k8s.IsomateConfig
		chartWalmVersion  common.WalmVersion
	}{
		{
			releaseNamespace: "testReleaseNamespace",
			releaseName:      "testReleaseName",
			repo:             "testRepo",
			chartName:        "testChartName",
			chartVersion:     "5.2",
			chartAppVersion:  "5.2",
			chartWalmVersion: common.WalmVersionV2,
		},
	}

	for _, test := range tests {
		_, err := buildAutoGenReleaseConfig(
			test.releaseNamespace, test.releaseName, test.repo, test.chartName,
			test.chartVersion, test.chartAppVersion, test.labels, test.dependencies,
			test.dependencyConfigs, test.userConfigs, test.chartImage, test.isomateConfig, test.chartWalmVersion,
		)
		assert.Nil(t, err)
	}
}
