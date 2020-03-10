package redis

import (
	"WarpCloud/walm/pkg/models/project"
	"WarpCloud/walm/pkg/redis"
	"encoding/json"
	"k8s.io/klog"
)

type Cache struct {
	redis redis.Redis
}

func (cache *Cache) GetProjectTask(namespace, name string) (projectTask *project.ProjectTask, err error) {
	projectTaskStr, err := cache.redis.GetFieldValue(redis.WalmProjectsKey, namespace, name)
	if err != nil {
		return nil, err
	}

	projectTask = &project.ProjectTask{}
	err = json.Unmarshal([]byte(projectTaskStr), projectTask)
	if err != nil {
		klog.Errorf("failed to unmarshal projectTaskStr %s : %s", projectTaskStr, err.Error())
		return nil, err
	}
	projectTask.CompatiblePreviousProjectTask()
	return
}

func (cache *Cache) GetProjectTasks(namespace string) (projectTasks []*project.ProjectTask, err error) {
	projectTaskStrs, err := cache.redis.GetFieldValues(redis.WalmProjectsKey, namespace, "")
	if err != nil {
		return nil, err
	}

	projectTasks = []*project.ProjectTask{}
	for _, projectTaskStr := range projectTaskStrs {
		projectTask := &project.ProjectTask{}

		err = json.Unmarshal([]byte(projectTaskStr), projectTask)
		if err != nil {
			klog.Errorf("failed to unmarshal project task of %s: %s", projectTaskStr, err.Error())
			return nil, err
		}
		projectTask.CompatiblePreviousProjectTask()
		projectTasks = append(projectTasks, projectTask)
	}

	return
}

func (cache *Cache) CreateOrUpdateProjectTask(projectTask *project.ProjectTask) error {
	if projectTask == nil {
		klog.Warning("failed to create or update project task as it is nil")
		return nil
	}
	klog.Infof("start to set project task of %s/%s to redis", projectTask.Namespace, projectTask.Name)
	err := cache.redis.SetFieldValues(redis.WalmProjectsKey, map[string]interface{}{redis.BuildFieldName(projectTask.Namespace, projectTask.Name): projectTask})
	if err != nil {
		return err
	}
	klog.Infof("succeed to set project task of %s/%s to redis", projectTask.Namespace, projectTask.Name)
	return nil
}

func (cache *Cache) DeleteProjectTask(namespace, name string) error {
	err := cache.redis.DeleteField(redis.WalmProjectsKey, namespace, name)
	if err != nil {
		return err
	}
	return nil
}

func NewProjectCache(redis redis.Redis) *Cache {
	return &Cache{
		redis: redis,
	}
}
