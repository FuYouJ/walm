package redis

import (
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/redis"
	"encoding/json"
	"k8s.io/klog"
	"strings"
	"time"
)

type Cache struct {
	redis redis.Redis
}

func (cache *Cache) GetReleaseCache(namespace, name string) (releaseCache *release.ReleaseCache, err error) {
	releaseCacheStr, err := cache.redis.GetFieldValue(redis.WalmReleasesKey, namespace, name)
	if err != nil {
		return
	}

	releaseCache = &release.ReleaseCache{}
	err = json.Unmarshal([]byte(releaseCacheStr), releaseCache)
	if err != nil {
		klog.Errorf("failed to unmarshal release cache of %s: %s", name, err.Error())
		return
	}
	return
}

func (cache *Cache) GetReleaseCaches(namespace, filter string) (releaseCaches []*release.ReleaseCache, err error) {
	releaseCacheStrs, err := cache.redis.GetFieldValues(redis.WalmReleasesKey, namespace, filter)
	if err != nil {
		return nil, err
	}

	releaseCaches = []*release.ReleaseCache{}
	for _, releaseCacheStr := range releaseCacheStrs {
		releaseCache := &release.ReleaseCache{}

		err = json.Unmarshal([]byte(releaseCacheStr), releaseCache)
		if err != nil {
			klog.Errorf("failed to unmarshal release cache of %s: %s", releaseCacheStr, err.Error())
			return
		}
		releaseCaches = append(releaseCaches, releaseCache)
	}

	return
}

func (cache *Cache) GetReleaseCachesByReleaseConfigs(releaseConfigs []*k8s.ReleaseConfig) (releaseCaches []*release.ReleaseCache, error error) {
	releaseCaches = []*release.ReleaseCache{}
	if len(releaseConfigs) == 0 {
		return
	}

	releaseCacheFieldNames := []string{}
	for _, releaseConfig := range releaseConfigs {
		releaseCacheFieldNames = append(releaseCacheFieldNames, redis.BuildFieldName(releaseConfig.Namespace, releaseConfig.Name))
	}

	releaseCacheStrs, err := cache.redis.GetFieldValuesByNames(redis.WalmReleasesKey, releaseCacheFieldNames...)
	if err != nil {
		return nil, err
	}

	for index, releaseCacheStr := range releaseCacheStrs {
		if releaseCacheStr == "" {
			klog.Warningf("release cache %s is not found", releaseCacheFieldNames[index])
			continue
		}

		releaseCache := &release.ReleaseCache{}
		err = json.Unmarshal([]byte(releaseCacheStr), releaseCache)
		if err != nil {
			klog.Errorf("failed to unmarshal release cache of %s: %s", releaseCacheStr, err.Error())
			return nil, err
		}
		releaseCaches = append(releaseCaches, releaseCache)
	}

	return
}

func (cache *Cache) CreateOrUpdateReleaseCache(releaseCache *release.ReleaseCache) error {
	if releaseCache == nil {
		klog.Warningf("failed to create or update cache as release cache is nil")
		return nil
	}

	err := cache.redis.SetFieldValues(redis.WalmReleasesKey, map[string]interface{}{redis.BuildFieldName(releaseCache.Namespace, releaseCache.Name): releaseCache})
	if err != nil {
		return err
	}
	klog.V(2).Infof("succeed to set release cache of %s/%s to redis", releaseCache.Namespace, releaseCache.Name)
	return nil
}

func (cache *Cache) DeleteReleaseCache(namespace string, name string) error {
	err := cache.redis.DeleteField(redis.WalmReleasesKey, namespace, name)
	if err != nil {
		return err
	}
	klog.V(2).Infof("succeed to delete release cache of %s from redis", name)
	return nil
}

func (cache *Cache) GetReleaseTask(namespace, name string) (releaseTask *release.ReleaseTask, err error) {
	releaseTaskStr, err := cache.redis.GetFieldValue(redis.WalmReleaseTasksKey, namespace, name)
	if err != nil {
		return nil, err
	}

	releaseTask = &release.ReleaseTask{}
	err = json.Unmarshal([]byte(releaseTaskStr), releaseTask)
	if err != nil {
		klog.Errorf("failed to unmarshal releaseTaskStr %s : %s", releaseTaskStr, err.Error())
		return nil, err
	}
	return
}

func (cache *Cache) GetReleaseTasks(namespace, filter string) (releaseTasks []*release.ReleaseTask, err error) {
	releaseTaskStrs, err := cache.redis.GetFieldValues(redis.WalmReleaseTasksKey, namespace, filter)
	if err != nil {
		return nil, err
	}

	releaseTasks = []*release.ReleaseTask{}
	for _, releaseTaskStr := range releaseTaskStrs {
		releaseTask := &release.ReleaseTask{}

		err = json.Unmarshal([]byte(releaseTaskStr), releaseTask)
		if err != nil {
			klog.Errorf("failed to unmarshal release task of %s: %s", releaseTaskStr, err.Error())
			return nil, err
		}
		releaseTasks = append(releaseTasks, releaseTask)
	}

	return
}

func (cache *Cache) GetReleaseTasksByReleaseConfigs(releaseConfigs []*k8s.ReleaseConfig) (releaseTasks []*release.ReleaseTask, err error) {
	releaseTasks = []*release.ReleaseTask{}
	if len(releaseConfigs) == 0 {
		return
	}

	releaseTaskFieldNames := []string{}
	for _, releaseConfig := range releaseConfigs {
		releaseTaskFieldNames = append(releaseTaskFieldNames, redis.BuildFieldName(releaseConfig.Namespace, releaseConfig.Name))
	}

	releaseTaskStrs, err := cache.redis.GetFieldValuesByNames(redis.WalmReleaseTasksKey, releaseTaskFieldNames...)
	if err != nil {
		return nil, err
	}

	for index, releaseTaskStr := range releaseTaskStrs {
		if releaseTaskStr == "" {
			klog.Warningf("release task %s is not found", releaseTaskFieldNames[index])
			continue
		}

		releaseTask := &release.ReleaseTask{}

		err = json.Unmarshal([]byte(releaseTaskStr), releaseTask)
		if err != nil {
			klog.Errorf("failed to unmarshal release task of %s: %s", releaseTaskStr, err.Error())
			return nil, err
		}
		releaseTasks = append(releaseTasks, releaseTask)
	}

	return
}

func (cache *Cache) CreateOrUpdateReleaseTask(releaseTask *release.ReleaseTask) error {
	if releaseTask == nil {
		klog.Warning("failed to create or update release task as it is nil")
		return nil
	}

	err := cache.redis.SetFieldValues(redis.WalmReleaseTasksKey, map[string]interface{}{redis.BuildFieldName(releaseTask.Namespace, releaseTask.Name): releaseTask})
	if err != nil {
		return err
	}
	klog.V(2).Infof("succeed to set release task of %s/%s to redis", releaseTask.Namespace, releaseTask.Name)
	return nil
}

func (cache *Cache) DeleteReleaseTask(namespace string, name string) error {
	err := cache.redis.DeleteField(redis.WalmReleaseTasksKey, namespace, name)
	if err != nil {
		return err
	}
	klog.V(2).Infof("succeed to delete release task of %s from redis", name)
	return nil
}

func (cache *Cache) CreateReleaseBackUp(namespace string, name string, releaseInfoByte []byte) error {
	key := redis.BuildMixedTopKey(redis.WalmReleasesKey, redis.BuildFieldName(namespace, name))
	err := cache.redis.SetKeyWithTTL(key, releaseInfoByte, time.Hour * 24 * 7)
	if err != nil {
		return err
	}
	return nil
}

func(cache *Cache) GetReleaseBackUp(namespace string, name string) (*release.ReleaseInfoV2, error) {
	value, err := cache.redis.GetValue(redis.BuildMixedTopKey(redis.WalmReleasesKey, redis.BuildFieldName(namespace, name)))
	if err != nil {
		return nil, err
	}
	releaseInfoV2 := &release.ReleaseInfoV2{}
	err = json.Unmarshal([]byte(value), releaseInfoV2)
	if err != nil {
		return nil, err
	}
	return releaseInfoV2, nil
}

func(cache *Cache) ListReleasesBackUp(namespace string) ([]*release.ReleaseInfoV2, error) {
	var releaseInfoV2List []*release.ReleaseInfoV2
	regex := ""
	if namespace != "" {
		regex = redis.BuildMixedTopKey(redis.WalmReleasesKey, namespace + "/*")
	} else {
		regex = redis.BuildMixedTopKey(redis.WalmReleasesKey, "*")
	}
	keys, err := cache.redis.GetKeys(regex)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		tokens := strings.FieldsFunc(key, func(r rune) bool {
			return r == '/' || r == '}'
		})
		releaseInfoV2, err := cache.GetReleaseBackUp(tokens[1], tokens[2])
		if err != nil {
			return nil, err
		}
		releaseInfoV2List = append(releaseInfoV2List, releaseInfoV2)
	}
	return releaseInfoV2List, nil
}

func NewCache(redis redis.Redis) *Cache {
	return &Cache{
		redis: redis,
	}
}
