package utils

import (
	"fmt"
	"k8s.io/klog"
	"reflect"
	"strings"
)

const (
	ReleaseSep = "/"
	CompatibleReleaseSep = "."
)

func ConfigValuesDiff(configValue1 map[string]interface{}, configValue2 map[string]interface{}) bool {
	if len(configValue1) == 0 && len(configValue2) == 0 {
		return false
	}
	return !reflect.DeepEqual(configValue1, configValue2)
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

