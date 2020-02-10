package release

import "WarpCloud/walm/pkg/models/common"

type RepoInfo struct {
	TenantRepoName string `json:"repoName"`
	TenantRepoURL  string `json:"repoUrl"`
}

type RepoInfoList struct {
	Items []*RepoInfo `json:"items" description:"chart repo list"`
}

type ChartInfo struct {
	ChartName        string         `json:"chartName"`
	ChartVersion     string         `json:"chartVersion"`
	ChartDescription string         `json:"chartDescription"`
	ChartAppVersion  string         `json:"chartAppVersion"`
	ChartEngine      string         `json:"chartEngine"`
	DefaultValue     string         `json:"defaultValue" description:"default values.yaml defined by the chart"`
	MetaInfo         *ChartMetaInfo `json:"metaInfo,omitempty" description:"transwarp chart meta info"`
	// Compatible
	DependencyCharts  []ChartDependencyInfo `json:"dependencyCharts,omitempty" description:"dependency chart name"`
	ChartPrettyParams *PrettyChartParams    `json:"chartPrettyParams,omitempty" description:"pretty chart params for market"`
	WalmVersion       common.WalmVersion    `json:"walmVersion" description:"chart walm version: v1, v2"`
}

type ChartDetailInfo struct {
	ChartInfo
	// additional info
	Advantage    string `json:"advantage" description:"chart production advantage description(rich text)"`
	Architecture string `json:"architecture" description:"chart production architecture description(rich text)"`
	Icon         string `json:"icon" description:"chart icon"`
	IconType     string `json:"iconType" description:"chart icon type"`
}

type ChartInfoList struct {
	Items []*ChartInfo `json:"items" description:"chart list"`
}
