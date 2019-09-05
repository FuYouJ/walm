package impl

import (
	"testing"
	"WarpCloud/walm/pkg/models/release"
	"github.com/stretchr/testify/assert"
	helmRelease "helm.sh/helm/pkg/release"
)

func Test_ReuseReleaseRequest(t *testing.T) {
	tests := []struct {
		releaseInfo    *release.ReleaseInfoV2
		releaseRequest *release.ReleaseRequestV2
		configValues   map[string]interface{}
		dependencies   map[string]string
		releaseLabels  map[string]string
		walmPlugins    []*release.ReleasePlugin
		err            error
	}{
		{
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						ConfigValues: map[string]interface{}{
							"existed-key": "old-value",
						},
						Dependencies: map[string]string{
							"existed-key": "old-value",
						},
					},
				},
				Plugins: []*release.ReleasePlugin{
					{
						Name: "existed-plugin",
						Args: "old-value",
					},
				},
				ReleaseLabels: map[string]string{
					"existed-key": "old-value",
				},
			},
			releaseRequest: &release.ReleaseRequestV2{},
			configValues: map[string]interface{}{
				"existed-key": "old-value",
			},
			dependencies: map[string]string{
				"existed-key": "old-value",
			},
			releaseLabels: map[string]string{
				"existed-key": "old-value",
			},
			walmPlugins: []*release.ReleasePlugin{
				{
					Name: "existed-plugin",
					Args: "old-value",
				},
			},
			err: nil,
		},
		{
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						ConfigValues: map[string]interface{}{
							"existed-key": "old-value",
						},
						Dependencies: map[string]string{
							"existed-key": "old-value",
						},
					},
				},
				Plugins: []*release.ReleasePlugin{
					{
						Name: "existed-plugin",
						Args: "old-value",
					},
				},
				ReleaseLabels: map[string]string{
					"existed-key": "old-value",
				},
			},
			releaseRequest: &release.ReleaseRequestV2{
				ReleaseRequest: release.ReleaseRequest{
					ConfigValues: map[string]interface{}{
						"existed-key":     "new-value",
						"not-existed-key": "value",
					},
					Dependencies: map[string]string{
						"existed-key":     "new-value",
						"not-existed-key": "value",
					},
				},
				ReleaseLabels: map[string]string{
					"existed-key":     "new-value",
					"not-existed-key": "value",
				},
				Plugins: []*release.ReleasePlugin{
					{
						Name: "existed-plugin",
						Args: "new-value",
					},
					{
						Name: "not-existed-plugin",
						Args: "value",
					},
				},
			},
			configValues: map[string]interface{}{
				"existed-key":     "new-value",
				"not-existed-key": "value",
			},
			dependencies: map[string]string{
				"existed-key":     "new-value",
				"not-existed-key": "value",
			},
			releaseLabels: map[string]string{
				"existed-key":     "new-value",
				"not-existed-key": "value",
			},
			walmPlugins: []*release.ReleasePlugin{
				{
					Name: "existed-plugin",
					Args: "new-value",
				},
				{
					Name: "not-existed-plugin",
					Args: "value",
				},
			},
			err: nil,
		},
		{
			releaseInfo: &release.ReleaseInfoV2{
				ReleaseInfo: release.ReleaseInfo{
					ReleaseSpec: release.ReleaseSpec{
						ConfigValues: map[string]interface{}{
							"existed-key": "old-value",
						},
						Dependencies: map[string]string{
							"existed-key": "old-value",
						},
					},
				},
				Plugins: []*release.ReleasePlugin{
					{
						Name: "existed-plugin",
						Args: "old-value",
					},
				},
				ReleaseLabels: map[string]string{
					"existed-key": "old-value",
				},
			},
			releaseRequest: &release.ReleaseRequestV2{
				ReleaseRequest: release.ReleaseRequest{
					ConfigValues: map[string]interface{}{
						"existed-key": nil,
					},
					Dependencies: map[string]string{
						"existed-key": "",
					},
				},
				ReleaseLabels: map[string]string{
					"existed-key": "",
				},
				Plugins: []*release.ReleasePlugin{
					{
						Name:    "existed-plugin",
						Args:    "",
						Disable: true,
					},
				},
			},
			configValues: map[string]interface{}{
				"existed-key": nil,
			},
			dependencies: map[string]string{
			},
			releaseLabels: map[string]string{
			},
			walmPlugins: []*release.ReleasePlugin{
				{
					Name:    "existed-plugin",
					Args:    "",
					Disable: true,
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		configValues, dependencies, releaseLabels, walmPlugins, err := reuseReleaseRequest(test.releaseInfo, test.releaseRequest)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.configValues, configValues)
		assert.Equal(t, test.dependencies, dependencies)
		assert.Equal(t, test.releaseLabels, releaseLabels)
		assert.ElementsMatch(t, test.walmPlugins, walmPlugins)
	}
}

func Test_MergeReleasePlugins(t *testing.T) {
	tests := []struct {
		plugins        []*release.ReleasePlugin
		defaultPlugins []*release.ReleasePlugin
		mergedPlugins  []*release.ReleasePlugin
		err            error
	}{
		{
			plugins: []*release.ReleasePlugin{
				{
					Name: "test",
				},
				{
					Name: "test",
				},
			},
			mergedPlugins: nil,
			err:           buildDuplicatedPluginError("test"),
		},
	}
	for _, test := range tests {
		plugins, err := mergeReleasePlugins(test.plugins, test.defaultPlugins)
		assert.IsType(t, test.err, err)
		assert.ElementsMatch(t, test.mergedPlugins, plugins)
	}
}

func Test_filterHelmReleases(t *testing.T) {
	tests := []struct {
		releases         []*helmRelease.Release
		filteredReleases map[string]*helmRelease.Release
	}{
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
				{
					Namespace: "testns",
					Name: "rel2",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
				"testns/rel2" : {
					Namespace: "testns",
					Name: "rel2",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusPendingUpgrade,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusPendingUpgrade,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusPendingUpgrade,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusPendingUpgrade,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusFailed,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusPendingUpgrade,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusFailed,
					},
				},
			},
		},
		{
			releases: []*helmRelease.Release{
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusPendingUpgrade,
					},
				},
				{
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusFailed,
					},
				},
			},
			filteredReleases: map[string]*helmRelease.Release{
				"testns/rel1" : {
					Namespace: "testns",
					Name: "rel1",
					Version: 2,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusFailed,
					},
				},
			},
		},
	}
	for _, test := range tests {
		filteredReleases := filterHelmReleases(test.releases)
		assert.Equal(t, test.filteredReleases, filteredReleases)
	}
}
