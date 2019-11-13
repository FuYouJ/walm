package utils

import (
	"fmt"
	"k8s.io/klog"
	"reflect"
	"strings"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/util"
)

const (
	ReleaseSep           = "/"
	CompatibleReleaseSep = "."
)

func ConfigValuesDiff(configValue1 map[string]interface{}, configValue2 map[string]interface{}) bool {
	if len(configValue1) == 0 && len(configValue2) == 0 {
		return false
	}
	unifiedConfigValue1, _ := util.UnifyConfigValue(configValue1)
	unifiedConifgValue2, _ := util.UnifyConfigValue(configValue2)
	return !reflect.DeepEqual(unifiedConfigValue1, unifiedConifgValue2)
}

func ParseDependedRelease(dependingReleaseNamespace, dependedRelease string) (namespace, name string, err error) {
	//Compatible
	compatibleDependedRelease := strings.Replace(dependedRelease, CompatibleReleaseSep, ReleaseSep, -1)
	ss := strings.Split(compatibleDependedRelease, ReleaseSep)
	if len(ss) == 2 {
		namespace = ss[0]
		name = ss[1]
	} else if len(ss) == 1 {
		namespace = dependingReleaseNamespace
		name = ss[0]
	} else {
		err = fmt.Errorf("depended release %s is not valid: only 1 or 0 seperator (%s or %s) is allowed", dependedRelease, ReleaseSep, CompatibleReleaseSep)
		klog.Warning(err.Error())
		return
	}
	return
}

func ConvertReleaseConfigDataFromRelease(r *release.ReleaseInfoV2) *release.ReleaseConfigData {
	return &release.ReleaseConfigData{
		ReleaseConfig: k8s.ReleaseConfig{
			Meta:                     k8s.NewMeta(k8s.ReleaseConfigKind, r.Namespace, r.Name, k8s.NewState("", "", "")),
			Labels:                   r.ReleaseLabels,
			OutputConfig:             r.OutputConfigValues,
			ChartImage:               r.ChartImage,
			ChartName:                r.ChartName,
			ConfigValues:             r.ConfigValues,
			Dependencies:             r.Dependencies,
			ChartVersion:             r.ChartVersion,
			ChartAppVersion:          r.ChartAppVersion,
			Repo:                     r.RepoName,
			DependenciesConfigValues: r.DependenciesConfigValues,
			IsomateConfig:            r.IsomateConfig,
			ChartWalmVersion:         r.ChartWalmVersion,
		},
		ReleaseWalmVersion: r.ReleaseWarmVersion,
	}
}

func ConvertReleaseConfigDatasFromReleaseList(rList []*release.ReleaseInfoV2) (cList *release.ReleaseConfigDataList) {
	cList = &release.ReleaseConfigDataList{}
	for _, r := range rList {
		cList.Items = append(cList.Items, ConvertReleaseConfigDataFromRelease(r))
	}
	cList.Num = len(cList.Items)
	return
}
