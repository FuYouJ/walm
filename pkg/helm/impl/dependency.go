package impl

import (
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/common"
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strings"
	"WarpCloud/walm/pkg/release/utils"
	"k8s.io/klog"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	errorModel "WarpCloud/walm/pkg/models/error"
	k8sutils "WarpCloud/walm/pkg/k8s/utils"
)

const (
	DependencyStatementReg       string = "^\\$\\(\\w+\\)(\\.\\w+)*$"
	DependencyStatementVarKeyReg string = "\\$\\(\\w+\\)"
)

// release with v1 chart should depend on release with v1 chart
// release with v2 chart should depend on release with v2 chart
func (helmImpl *Helm) GetDependencyOutputConfigs(
	namespace string, dependencies map[string]string,
	chartInfo *release.ChartDetailInfo, strict bool,
) (dependencyConfigs map[string]interface{}, err error) {
	if chartInfo.WalmVersion == common.WalmVersionV2 {
		return helmImpl.getDependencyOutputConfigsForChartV2(namespace, dependencies, chartInfo, strict)
	} else if chartInfo.WalmVersion == common.WalmVersionV1 {
		return helmImpl.getDependencyOutputConfigsForChartV1(namespace, dependencies, chartInfo, strict)
	}
	return nil, nil
}

func (helmImpl *Helm) getDependencyOutputConfigsForChartV1(
	namespace string, dependencies map[string]string,
	chartInfo *release.ChartDetailInfo,
	strict bool,
) (dependencyConfigs map[string]interface{}, err error) {
	dependencyConfigs = map[string]interface{}{}

	// compatible v1 chart
	dependencyRequires := map[string]map[string]string{}
	for _, dependencyChart := range chartInfo.DependencyCharts {
		dependencyRequires[dependencyChart.ChartName] = dependencyChart.Requires
	}

	for dependencyKey, dependency := range dependencies {
		dependencyRequire, ok := dependencyRequires[dependencyKey]
		if !ok {
			err = fmt.Errorf("dependency key %s is not valid", dependencyKey)
			klog.Errorf(err.Error())
			return
		}

		dependencyMeta, err := helmImpl.getDependencyMetaForChartV1(namespace, dependency)
		if err != nil {
			if errorModel.IsNotFoundError(err) && !strict {
				klog.Warningf("ignore dependency not found error due to disable strict mode : %s", err.Error())
			} else {
				klog.Errorf("failed to get dependency meta : %s", err.Error())
				return nil, err
			}
		}
		if dependencyMeta == nil {
			continue
		}

		err = buildDependencyConfigsForChartV1(dependencyConfigs, dependencyRequire, dependencyMeta)
		if err != nil {
			return nil, err
		}
	}
	return dependencyConfigs, nil
}

func buildDependencyConfigsForChartV1(
	dependencyConfigs map[string]interface{},
	dependencyRequire map[string]string, dependencyMeta *k8sModel.DependencyMeta,
) error {
	cache := make(map[string]interface{})
	for key, statement := range dependencyRequire {
		varName, fieldPath, err := splitVarAndFieldPath(statement)
		if err != nil {
			return err
		}
		klog.V(2).Infof("varName \"%s\", field path \"%s\"", varName, fieldPath)
		varValue, found := cache[varName]
		if !found {
			varValue, err = getProvidedValue(dependencyMeta, varName)
			if err != nil {
				return err
			}
			cache[varName] = varValue
		}
		fieldValue, err := getFieldPathValue(varValue, fieldPath)
		if err != nil {
			klog.Errorf("Fail to get value for %s, error %v", varName, err)
			return err
		}
		dependencyConfigs[key] = fieldValue
	}
	return nil
}

func (helmImpl *Helm) getDependencyOutputConfigsForChartV2(namespace string, dependencies map[string]string, chartInfo *release.ChartDetailInfo, strict bool) (dependencyConfigs map[string]interface{}, err error) {
	dependencyConfigs = map[string]interface{}{}
	dependencyAliasConfigVars := map[string]string{}
	if chartInfo.MetaInfo == nil {
		return
	}
	chartDependencies := chartInfo.MetaInfo.ChartDependenciesInfo
	for _, chartDependency := range chartDependencies {
		dependencyAliasConfigVars[chartDependency.Name] = chartDependency.AliasConfigVar
	}

	for dependencyKey, dependency := range dependencies {
		dependencyAliasConfigVar, ok := dependencyAliasConfigVars[dependencyKey]
		if !ok {
			err = fmt.Errorf("dependency key %s is not valid, you can see valid keys in chart metainfo", dependencyKey)
			klog.Errorf(err.Error())
			return
		}

		outputConfig, err := helmImpl.getOutputConfigValuesForChartV2(namespace, dependency)
		if err != nil {
			if errorModel.IsNotFoundError(err) && !strict {
				klog.Warningf("ignore dependency not found error due to disable strict mode : %s", err.Error())
			} else {
				klog.Errorf("failed to get dependency %s output config value : %s", dependency, err.Error())
				return nil, err
			}
		}

		if len(outputConfig) > 0 {
			dependencyConfigs[dependencyAliasConfigVar] = outputConfig
		}
	}
	return
}

func (helmImpl *Helm) getDependencyMetaForChartV1(namespace string, dependency string) (*k8sModel.DependencyMeta, error) {
	dependencyNamespace, dependencyName, err := utils.ParseDependedRelease(namespace, dependency)
	if err != nil {
		return nil, err
	}

	// compatible v1 release
	dependencyInstanceResource, err := helmImpl.k8sCache.GetResource(k8sModel.InstanceKind, dependencyNamespace, dependencyName)
	if err != nil {
		if !errorModel.IsNotFoundError(err) {
			klog.Errorf("failed to get instance %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
			return nil, err
		}
	} else {
		dependencyInstance := dependencyInstanceResource.(*k8sModel.ApplicationInstance)
		return dependencyInstance.DependencyMeta, nil
	}

	dependencyReleaseConfigResource, err := helmImpl.k8sCache.GetResource(k8sModel.ReleaseConfigKind, dependencyNamespace, dependencyName)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Errorf("release config %s/%s is not found", dependencyNamespace, dependencyName)
			return nil, errors.Wrapf(err, "release config %s/%s is not found", dependencyNamespace, dependencyName)
		}
		klog.Errorf("failed to get release config %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
		return nil, err
	}

	dependencyReleaseConfig := dependencyReleaseConfigResource.(*k8sModel.ReleaseConfig)
	return k8sutils.ConvertOutputConfigToDependencyMeta(dependencyReleaseConfig.OutputConfig), nil
}

func (helmImpl *Helm) getOutputConfigValuesForChartV2(namespace string, dependency string) (map[string]interface{}, error) {
	dependencyNamespace, dependencyName, err := utils.ParseDependedRelease(namespace, dependency)
	if err != nil {
		return nil, err
	}

	// compatible v1 release
	dependencyInstanceResource, err := helmImpl.k8sCache.GetResource(k8sModel.InstanceKind, dependencyNamespace, dependencyName)
	if err != nil {
		if !errorModel.IsNotFoundError(err) {
			klog.Errorf("failed to get instance %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
			return nil, err
		}
	} else {
		dependencyMeta := (dependencyInstanceResource.(*k8sModel.ApplicationInstance)).DependencyMeta
		return k8sutils.ConvertDependencyMetaToOutputConfig(dependencyMeta), nil
	}

	dependencyReleaseConfigResource, err := helmImpl.k8sCache.GetResource(k8sModel.ReleaseConfigKind, dependencyNamespace, dependencyName)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Errorf("release config %s/%s is not found", dependencyNamespace, dependencyName)
			return nil, errors.Wrapf(err, "release config %s/%s is not found", dependencyNamespace, dependencyName)
		}
		klog.Errorf("failed to get release config %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
		return nil, err
	}

	dependencyReleaseConfig := dependencyReleaseConfigResource.(*k8sModel.ReleaseConfig)
	return dependencyReleaseConfig.OutputConfig, nil
}

// split varialbe and field path from the dependency statement
// eg. "$(ZK_RC).metadata.name" -> "ZK_RC", "metadata.name"
func splitVarAndFieldPath(statement string) (string, string, error) {
	if !regexp.MustCompile(DependencyStatementReg).MatchString(statement) {
		return "", "", fmt.Errorf("Invalid statement, does not match %s", DependencyStatementReg)
	}

	varKey := regexp.MustCompile(DependencyStatementVarKeyReg).FindString(statement)
	if len(statement) == len(varKey) {
		return varKey[2 : len(varKey)-1], "", nil
	} else {
		return varKey[2 : len(varKey)-1], statement[len(varKey)+1:], nil
	}

}

// getProvidedValue helps to extract instance's provided value with the help of dependency meta in dummy service's annotation
func getProvidedValue(meta *k8sModel.DependencyMeta, varName string) (interface{}, error) {
	for name, provide := range meta.Provides {
		if name == varName {
			if provide.ResourceType == "" {
				return provide.ImmediateValue, nil
			}
		}
	}
	err := fmt.Errorf("failed to get provided value %s", varName)
	klog.Errorf(err.Error())
	return nil, err
}

// getFieldPathValue helps to get value for field path
// eg. rc.spec.replicas
func getFieldPathValue(configs interface{}, path string) (interface{}, error) {
	if path == "" {
		return configs, nil
	}
	fields := strings.Split(path, ".")
	if len(fields) == 0 {
		return configs, nil
	}

	fieldValue := configs
	parsedField := make([]string, 0)
	for _, field := range fields {
		parsedField = append(parsedField, field)
		fieldMap, ok := fieldValue.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Fail to parse field path %s", strings.Join(parsedField, "."))
		}
		value, found := fieldMap[field]
		if !found {
			return nil, fmt.Errorf("Fail to get field path value %s", strings.Join(parsedField, "."))
		} else {
			fieldValue = value
		}
	}
	return fieldValue, nil
}
