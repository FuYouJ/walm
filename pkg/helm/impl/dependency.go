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
	walmutils "WarpCloud/walm/pkg/util"
)

const (
	DependencyStatementReg       string = "^\\$\\(\\w+\\)(\\.\\w+)*$"
	DependencyStatementVarKeyReg string = "\\$\\(\\w+\\)"

	dummyDependencyProvideKey = "dummyDependencyProvideKey"
)

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
			delete(dependencies, dependencyKey)
			klog.Warningf("dependency key %s is not valid, ignore error", dependencyKey)
			continue
		}

		dependencyChartWalmVersion, dependencyMeta, err := helmImpl.getDependencyMetaForChartV1(namespace, dependency)
		if err != nil {
			if errorModel.IsNotFoundError(err) && !strict {
				klog.Warningf("ignore dependency not found error due to disable strict mode : %s", err.Error())
			} else {
				klog.Errorf("failed to get dependency meta : %s", err.Error())
				return dependencyConfigs, err
			}
		}
		if dependencyMeta == nil || len(dependencyMeta.Provides) == 0 {
			continue
		}

		err = buildDependencyConfigsForChartV1(dependencyConfigs, dependencyRequire, dependencyMeta, dependencyChartWalmVersion)
		if err != nil {
			return nil, err
		}
	}
	return dependencyConfigs, nil
}

func buildDependencyConfigsForChartV1(dependencyConfigs map[string]interface{}, dependencyRequire map[string]string,
	dependencyMeta *k8sModel.DependencyMeta, dependencyChartWalmVersion common.WalmVersion) error {
	cache := make(map[string]interface{})
	for key, statement := range dependencyRequire {
		varName, fieldPath, err := splitVarAndFieldPath(statement)
		if err != nil {
			return err
		}
		klog.V(2).Infof("varName \"%s\", field path \"%s\"", varName, fieldPath)
		varValue, found := cache[varName]
		if !found {
			if dependencyChartWalmVersion == common.WalmVersionV1 {
				varValue, err = getProvidedValue(dependencyMeta, varName)
				if err != nil {
					return err
				}
				cache[varName] = varValue
			} else if dependencyChartWalmVersion == common.WalmVersionV2 {
				varValue, err = walmutils.ConvertObjectToJsonMap(dependencyMeta.Provides[dummyDependencyProvideKey].ImmediateValue)
				if err != nil {
					return err
				}
				cache[varName] = varValue
			}
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
			delete(dependencies, dependencyKey)
			klog.Warningf("dependency key %s is not valid, ignore error", dependencyKey)
			continue
		}

		dependencyChartWalmVersion, outputConfig, err := helmImpl.getOutputConfigValuesForChartV2(namespace, dependency)
		if err != nil {
			if errorModel.IsNotFoundError(err) && !strict {
				klog.Warningf("ignore dependency not found error due to disable strict mode : %s", err.Error())
				continue
			} else {
				klog.Errorf("failed to get dependency %s output config value : %s", dependency, err.Error())
				return dependencyConfigs, err
			}
		}

		if len(outputConfig) > 0 {
			if dependencyChartWalmVersion == common.WalmVersionV2 {
				dependencyConfigs[dependencyAliasConfigVar] = outputConfig
			} else if dependencyChartWalmVersion == common.WalmVersionV1 {
				dependencyMeta := k8sutils.ConvertOutputConfigToDependencyMeta(outputConfig)
				if dependencyMeta != nil && dependencyMeta.Provides != nil {
					dependencyConfigs[dependencyAliasConfigVar] = dependencyMeta.Provides[dependencyAliasConfigVar].ImmediateValue
				}
			}
		}
	}
	return dependencyConfigs, nil
}

func (helmImpl *Helm) getDependencyMetaForChartV1(namespace string, dependency string) (common.WalmVersion, *k8sModel.DependencyMeta, error) {
	dependencyNamespace, dependencyName, err := utils.ParseDependedRelease(namespace, dependency)
	if err != nil {
		return "", nil, err
	}

	// compatible v1 release
	dependencyInstanceResource, err := helmImpl.k8sCache.GetResource(k8sModel.InstanceKind, dependencyNamespace, dependencyName)
	if err != nil {
		if !errorModel.IsNotFoundError(err) {
			klog.Errorf("failed to get instance %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
			return "", nil, err
		}
	} else {
		dependencyInstance := dependencyInstanceResource.(*k8sModel.ApplicationInstance)
		return common.WalmVersionV1, dependencyInstance.DependencyMeta, nil
	}

	dependencyReleaseConfigResource, err := helmImpl.k8sCache.GetResource(k8sModel.ReleaseConfigKind, dependencyNamespace, dependencyName)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Errorf("release config %s/%s is not found", dependencyNamespace, dependencyName)
			return "", nil, errors.Wrapf(err, "release config %s/%s is not found", dependencyNamespace, dependencyName)
		}
		klog.Errorf("failed to get release config %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
		return "", nil, err
	}

	dependencyReleaseConfig := dependencyReleaseConfigResource.(*k8sModel.ReleaseConfig)
	chartWalmVersion := buildCompatibleChartWalmVersion(dependencyReleaseConfig)
	var dependencyMeta *k8sModel.DependencyMeta
	if chartWalmVersion == common.WalmVersionV1 {
		dependencyMeta = k8sutils.ConvertOutputConfigToDependencyMeta(dependencyReleaseConfig.OutputConfig)
	} else if chartWalmVersion == common.WalmVersionV2 {
		dependencyMeta = &k8sModel.DependencyMeta{
			Provides: map[string]k8sModel.DependencyProvide{
				dummyDependencyProvideKey: {
					ImmediateValue: dependencyReleaseConfig.OutputConfig,
				},
			},
		}
	}
	return chartWalmVersion, dependencyMeta, nil
}

func (helmImpl *Helm) getOutputConfigValuesForChartV2(namespace string, dependency string) (common.WalmVersion, map[string]interface{}, error) {
	dependencyNamespace, dependencyName, err := utils.ParseDependedRelease(namespace, dependency)
	if err != nil {
		return "", nil, err
	}

	// compatible v1 release
	dependencyInstanceResource, err := helmImpl.k8sCache.GetResource(k8sModel.InstanceKind, dependencyNamespace, dependencyName)
	if err != nil {
		if !errorModel.IsNotFoundError(err) {
			klog.Errorf("failed to get instance %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
			return "", nil, err
		}
	} else {
		dependencyMeta := (dependencyInstanceResource.(*k8sModel.ApplicationInstance)).DependencyMeta
		return common.WalmVersionV1, k8sutils.ConvertDependencyMetaToOutputConfig(dependencyMeta), nil
	}

	dependencyReleaseConfigResource, err := helmImpl.k8sCache.GetResource(k8sModel.ReleaseConfigKind, dependencyNamespace, dependencyName)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("release config %s/%s is not found", dependencyNamespace, dependencyName)
			return "", nil, errors.Wrapf(err, "release config %s/%s is not found", dependencyNamespace, dependencyName)
		}
		klog.Errorf("failed to get release config %s/%s : %s", dependencyNamespace, dependencyName, err.Error())
		return "", nil, err
	}

	dependencyReleaseConfig := dependencyReleaseConfigResource.(*k8sModel.ReleaseConfig)
	return buildCompatibleChartWalmVersion(dependencyReleaseConfig), dependencyReleaseConfig.OutputConfig, nil
}

func buildCompatibleChartWalmVersion(releaseConfig *k8sModel.ReleaseConfig) common.WalmVersion {
	if releaseConfig.ChartWalmVersion == "" {
		if len(releaseConfig.OutputConfig) == 0 {
			return ""
		}

		dependencyMeta := k8sutils.ConvertOutputConfigToDependencyMeta(releaseConfig.OutputConfig)
		if dependencyMeta != nil && len(dependencyMeta.Provides) > 0 {
			return common.WalmVersionV1
		}

		return common.WalmVersionV2
	} else {
		return releaseConfig.ChartWalmVersion
	}
}

// split varialbe and field path from the dependency statement
// eg. "$(ZK_RC).metadata.name" -> "ZK_RC", "metadata.name"
func splitVarAndFieldPath(statement string) (string, string, error) {
	if !regexp.MustCompile(DependencyStatementReg).MatchString(statement) {
		err := fmt.Errorf("Invalid statement, does not match %s", DependencyStatementReg)
		klog.Error(err.Error())
		return "", "", err
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
	if configs == nil || path == "" {
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
