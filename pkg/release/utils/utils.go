package utils

import (
	"fmt"
	"k8s.io/klog"
	"reflect"
	"strings"
)

const (
	ReleaseSep = "/"
)

func ConfigValuesDiff(configValue1 map[string]interface{}, configValue2 map[string]interface{}) bool {
	if len(configValue1) == 0 && len(configValue2) == 0 {
		return false
	}
	return !reflect.DeepEqual(configValue1, configValue2)
}

func ParseDependedRelease(dependingReleaseNamespace, dependedRelease string) (namespace, name string, err error) {
	ss := strings.Split(dependedRelease, ReleaseSep)
	if len(ss) == 2 {
		namespace = ss[0]
		name = ss[1]
	} else if len(ss) == 1 {
		namespace = dependingReleaseNamespace
		name = ss[0]
	} else {
		err = fmt.Errorf("depended release %s is not valid: only 1 or 0 \"%s\" is allowed", dependedRelease, ReleaseSep)
		klog.Warning(err.Error())
		return
	}
	return
}

