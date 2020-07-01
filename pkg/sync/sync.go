package sync

import (
	"WarpCloud/walm/pkg/models/common"
	"strings"
	"time"

	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"

	"WarpCloud/walm/pkg/helm"
	"WarpCloud/walm/pkg/k8s"
	errorModel "WarpCloud/walm/pkg/models/error"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/project"
	releaseModel "WarpCloud/walm/pkg/models/release"
	walmRedis "WarpCloud/walm/pkg/redis"
	"WarpCloud/walm/pkg/task"
)

const (
	resyncInterval time.Duration = 5 * time.Minute
)

type Sync struct {
	redisClient *redis.Client
	helm        helm.Helm
	k8sCache    k8s.Cache
	task        task.Task

	releaseCacheKey string
	releaseTaskKey  string
	projectTaskKey  string
}

func (sync *Sync) Start(stopCh <-chan struct{}) {
	klog.Infof("start to resync release cache every %v", resyncInterval)
	// first time should be sync successfully
	count := 0
	for {
		err := sync.Resync()
		if err != nil {
			time.Sleep(time.Second * 30)
			count++
			if count > 1 {
				break
			}
			continue
		}
		break
	}

	firstTime := true

	go wait.Until(func() {
		if firstTime {
			time.Sleep(resyncInterval)
			firstTime = false
		}
		if err := sync.Resync(); err != nil {
			klog.Errorf("failed to resync release cache now: %s", err.Error())
			panic(err)
 		}
	}, resyncInterval, stopCh)
}

func (sync *Sync) Resync() error{
	for {
		err := sync.redisClient.Watch(func(tx *redis.Tx) error {

			releaseCachesFromHelm, err := sync.helm.ListAllReleases()
			if err != nil {
				klog.Errorf("failed to get release caches from helm : %s", err.Error())
				return err
			}

			releaseCachesFromHelmMap, err := buildReleaseCachesFromHelmMap(releaseCachesFromHelm)
			if err != nil {
				klog.Errorf("failed to build release cache map : %s", err.Error())
				return err
			}

			releaseCacheKeysFromRedis, err := tx.HKeys(sync.releaseCacheKey).Result()
			if err != nil {
				klog.Errorf("failed to get release cache keys from redis: %s", err.Error())
				return err
			}
			releaseCacheKeysToDel := buildReleaseCacheKeysToDel(releaseCacheKeysFromRedis, releaseCachesFromHelmMap)

			releaseConfigs, err := sync.k8sCache.ListReleaseConfigs("", "")
			if err != nil {
				klog.Errorf("failed to list release configs : %s", err.Error())
				return err
			}
			projectTasksFromReleaseConfigs, err := buildProjectTasksFromReleaseConfigs(releaseConfigs)
			if err != nil {
				klog.Errorf("failed to build project tasks by release configs : %s", err.Error())
				return err
			}
			err = buildProjectTasksFromReleaseCaches(projectTasksFromReleaseConfigs, releaseCachesFromHelm)
			if err != nil {
				klog.Errorf("failed to build project tasks by release names compatible v1 : %s", err.Error())
				return err
			}
			projectTasksInRedis, err := tx.HGetAll(sync.projectTaskKey).Result()
			if err != nil {
				klog.Errorf("failed to get project tasks from redis: %s", err.Error())
				return err
			}
			projectTasksToDel, err := sync.buildProjectTasksToDel(projectTasksFromReleaseConfigs, projectTasksInRedis)
			if err != nil {
				return err
			}
			projectTasksToSet := buildProjectTasksToSet(projectTasksFromReleaseConfigs, projectTasksInRedis)

			releaseTasksFromHelm, err := buildReleaseTasksFromHelm(releaseCachesFromHelmMap)
			if err != nil {
				return err
			}
			releaseTaskInRedis, err := tx.HGetAll(sync.releaseTaskKey).Result()
			if err != nil {
				klog.Errorf("failed to get release tasks from redis: %s", err.Error())
				return err
			}

			releaseTasksToDel, err := sync.buildReleaseTasksToDel(releaseTasksFromHelm, releaseTaskInRedis)
			if err != nil {
				return err
			}
			releaseTasksToSet := buildReleaseTasksToSet(releaseTasksFromHelm, releaseTaskInRedis)

			_, err = tx.Pipelined(func(pipe redis.Pipeliner) error {
				if len(releaseCachesFromHelm) > 0 {
					pipe.HMSet(sync.releaseCacheKey, releaseCachesFromHelmMap)
				}
				if len(releaseCacheKeysToDel) > 0 {
					pipe.HDel(sync.releaseCacheKey, releaseCacheKeysToDel...)
				}
				if len(projectTasksToSet) > 0 {
					pipe.HMSet(sync.projectTaskKey, projectTasksToSet)
				}
				if len(projectTasksToDel) > 0 {
					pipe.HDel(sync.projectTaskKey, projectTasksToDel...)
				}
				if len(releaseTasksToSet) > 0 {
					pipe.HMSet(sync.releaseTaskKey, releaseTasksToSet)
				}
				if len(releaseTasksToDel) > 0 {
					pipe.HDel(sync.releaseTaskKey, releaseTasksToDel...)
				}
				return nil
			})
			return err
		}, sync.releaseCacheKey, sync.projectTaskKey, sync.releaseTaskKey)

		if err == redis.TxFailedErr {
			klog.Warning("resync release cache transaction failed, will retry after 5 seconds")
			time.Sleep(5 * time.Second)
		} else {
			if err != nil {
				klog.Errorf("failed to resync release caches: %s", err.Error())
			} else {
				klog.Info("succeed to resync release caches")
			}
			return err
		}
	}

}

func (sync *Sync) buildReleaseTasksToDel(releaseTasksFromHelm, releaseTaskInRedis map[string]string) ([]string, error) {
	releaseTasksToDel := []string{}
	for releaseTaskKey, releaseTaskStr := range releaseTaskInRedis {
		if _, ok := releaseTasksFromHelm[releaseTaskKey]; !ok {
			releaseTask := &releaseModel.ReleaseTask{}
			err := json.Unmarshal([]byte(releaseTaskStr), releaseTask)
			if err != nil {
				klog.Errorf("failed to unmarshal release task string %s : %s", releaseTaskStr, err.Error())
				return nil, err
			}

			taskState, err := sync.task.GetTaskState(releaseTask.LatestReleaseTaskSig)
			if err != nil {
				if errorModel.IsNotFoundError(err) {
					releaseTasksToDel = append(releaseTasksToDel, releaseTaskKey)
				} else {
					klog.Errorf("failed to get task state : %s", err.Error())
					return nil, err
				}
			} else if taskState.IsFinished() || taskState.IsTimeout() {
				releaseTasksToDel = append(releaseTasksToDel, releaseTaskKey)
			}
		}
	}
	return releaseTasksToDel, nil
}

func (sync *Sync) buildProjectTasksToDel(projectTasksFromReleaseConfigs map[string]string,
	projectTasksInRedis map[string]string) ([]string, error) {
	projectTasksToDel := []string{}
	for projectTaskKey, projectTaskStr := range projectTasksInRedis {
		if _, ok := projectTasksFromReleaseConfigs[projectTaskKey]; !ok {
			projectTask := &project.ProjectTask{}
			err := json.Unmarshal([]byte(projectTaskStr), projectTask)
			if err != nil {
				klog.Errorf("failed to unmarshal projectTaskStr %s : %s", projectTaskStr, err.Error())
				return nil, err
			}

			projectTask.CompatiblePreviousProjectTask()

			taskState, err := sync.task.GetTaskState(projectTask.LatestTaskSignature)
			if err != nil {
				if errorModel.IsNotFoundError(err) {
					projectTasksToDel = append(projectTasksToDel, projectTaskKey)
				} else {
					klog.Errorf("failed to get task state : %s", err.Error())
					return nil, err
				}
			} else if taskState.IsFinished() || taskState.IsTimeout() {
				projectTasksToDel = append(projectTasksToDel, projectTaskKey)
			}
		}
	}
	return projectTasksToDel, nil
}

func buildReleaseCachesFromHelmMap(caches []*releaseModel.ReleaseCache) (map[string]interface{}, error) {
	cacheMap := map[string]*releaseModel.ReleaseCache{}
	for _, cache := range caches {
		filedName := walmRedis.BuildFieldName(cache.Namespace, cache.Name)
		if existedRelease, ok := cacheMap[filedName]; ok {
			if existedRelease.Version < cache.Version {
				cacheMap[filedName] = cache
			}
		} else {
			cacheMap[filedName] = cache
		}

	}
	return convertReleaseCachesMapToStrMap(cacheMap)
}

func convertReleaseCachesMapToStrMap(releaseCaches map[string]*releaseModel.ReleaseCache) (convertedResult map[string]interface{}, err error) {
	if releaseCaches != nil {
		convertedResult = make(map[string]interface{}, len(releaseCaches))
		for key, value := range releaseCaches {
			valueBytes, err := json.Marshal(value)
			if err != nil {
				klog.Errorf("failed to marshal value : %s", err.Error())
				return nil, err
			}
			convertedResult[key] = valueBytes
		}
	}
	return
}

func buildReleaseTasksToSet(releaseTasksFromHelm map[string]string, releaseTaskInRedis map[string]string) map[string]interface{} {
	releaseTasksToSet := map[string]interface{}{}
	for releaseTaskKey, releaseTaskStr := range releaseTasksFromHelm {
		if _, ok := releaseTaskInRedis[releaseTaskKey]; !ok {
			releaseTasksToSet[releaseTaskKey] = releaseTaskStr
		}
	}
	return releaseTasksToSet
}

func buildProjectTasksToSet(projectTasksFromReleaseConfigs map[string]string, projectTasksInRedis map[string]string) map[string]interface{} {
	projectCachesToSet := map[string]interface{}{}
	for projectCacheKey, projectCacheStr := range projectTasksFromReleaseConfigs {
		if _, ok := projectTasksInRedis[projectCacheKey]; !ok {
			projectCachesToSet[projectCacheKey] = projectCacheStr
		}
	}
	return projectCachesToSet
}

func buildReleaseTasksFromHelm(releaseCachesFromHelm map[string]interface{}) (map[string]string, error) {
	releaseTasksFromHelm := map[string]string{}
	for releaseCacheKey, releaseCacheStr := range releaseCachesFromHelm {
		releaseCache := &releaseModel.ReleaseCache{}
		err := json.Unmarshal(releaseCacheStr.([]byte), releaseCache)
		if err != nil {
			klog.Errorf("failed to unmarshal release cache of %s: %s", releaseCacheKey, err.Error())
			return nil, err
		}

		releaseTaskStr, err := json.Marshal(&releaseModel.ReleaseTask{
			Namespace: releaseCache.Namespace,
			Name:      releaseCache.Name,
		})
		if err != nil {
			klog.Errorf("failed to marshal release task of %s/%s: %s", releaseCache.Namespace, releaseCache.Name, err.Error())
			return nil, err
		}
		releaseTasksFromHelm[walmRedis.BuildFieldName(releaseCache.Namespace, releaseCache.Name)] = string(releaseTaskStr)
	}
	return releaseTasksFromHelm, nil
}

func buildProjectTasksFromReleaseConfigs(releaseConfigs []*k8sModel.ReleaseConfig) (map[string]string, error) {
	projectTasksFromReleaseConfigs := map[string]string{}
	for _, releaseConfig := range releaseConfigs {
		if projectName, ok1 := releaseConfig.Labels[project.ProjectNameLabelKey]; ok1 {
			_, ok := projectTasksFromReleaseConfigs[walmRedis.BuildFieldName(releaseConfig.Namespace, projectName)]
			if !ok {
				projectTaskStr, err := json.Marshal(&project.ProjectTask{
					Namespace:   releaseConfig.Namespace,
					Name:        projectName,
					WalmVersion: common.WalmVersionV2,
				})
				if err != nil {
					klog.Errorf("failed to marshal project task of %s/%s: %s", releaseConfig.Namespace, projectName, err.Error())
					return nil, err
				}
				projectTasksFromReleaseConfigs[walmRedis.BuildFieldName(releaseConfig.Namespace, projectName)] = string(projectTaskStr)
			}
		}
	}
	return projectTasksFromReleaseConfigs, nil
}

func buildProjectTasksFromReleaseCaches(projectTasks map[string]string, releaseCaches []*releaseModel.ReleaseCache) error {
	for _, releaseCache := range releaseCaches {
		projectNameArray := strings.Split(releaseCache.Name, "--")
		if len(projectNameArray) == 2 {
			projectName := projectNameArray[0]
			if _, ok := projectTasks[walmRedis.BuildFieldName(releaseCache.Namespace, projectName)]; !ok {
				projectCacheStr, err := json.Marshal(&project.ProjectTask{
					Namespace:   releaseCache.Namespace,
					Name:        projectName,
					WalmVersion: common.WalmVersionV1,
				})
				if err != nil {
					logrus.Errorf("failed to marshal project cache of %s/%s: %s", releaseCache.Namespace, projectName, err.Error())
					return err
				}
				projectTasks[walmRedis.BuildFieldName(releaseCache.Namespace, projectName)] = string(projectCacheStr)
			}
		}
	}
	return nil
}

func buildReleaseCacheKeysToDel(releaseCacheKeysFromRedis []string, releaseCachesFromHelm map[string]interface{}) []string {
	releaseCacheKeysToDel := []string{}
	for _, releaseCacheKey := range releaseCacheKeysFromRedis {
		if _, ok := releaseCachesFromHelm[releaseCacheKey]; !ok {
			releaseCacheKeysToDel = append(releaseCacheKeysToDel, releaseCacheKey)
		}
	}
	return releaseCacheKeysToDel
}

func NewSync(redisClient *redis.Client, helm helm.Helm, k8sCache k8s.Cache, task task.Task, releaseCacheKey, releaseTaskKey, projectTaskKey string) *Sync {
	result := &Sync{
		redisClient: redisClient,
		helm:        helm,
		k8sCache:    k8sCache,
		task:        task,

		releaseCacheKey: releaseCacheKey,
		releaseTaskKey:  releaseTaskKey,
		projectTaskKey:  projectTaskKey,
	}
	if result.releaseCacheKey == "" {
		result.releaseCacheKey = walmRedis.WalmReleasesKey
	}
	if result.releaseTaskKey == "" {
		result.releaseTaskKey = walmRedis.WalmReleaseTasksKey
	}
	if result.projectTaskKey == "" {
		result.projectTaskKey = walmRedis.WalmProjectsKey
	}
	return result
}
