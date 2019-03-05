package helm

import (
	"testing"
	"fmt"
)

func Test_Repo_List(t *testing.T) {
	chartInfo, err := GetChartInfo("stable", "demo", "1.0.0")
	fmt.Printf("%+v meta %+v %v\n", chartInfo, *chartInfo.MetaInfo, err)
	for _, dependencyInfo := range chartInfo.MetaInfo.ChartDependenciesInfo {
		fmt.Printf("%v\n", *dependencyInfo)
	}
}
