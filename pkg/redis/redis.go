package redis

import "time"

const (
	WalmReleasesKey   = "walm-releases"
	WalmProjectsKey   = "walm-project-tasks"
	WalmReleaseTasksKey   = "walm-release-tasks"
	WalmConfigKey	= "walm-config"
)

type Redis interface {
	GetFieldValue(key, namespace, name string) (string, error)
	GetFieldValues(key, namespace, filter string) ([]string, error)
	GetFieldValuesByNames(key string, filedNames... string) ([]string, error)
	SetFieldValues(key string, fieldValues map[string]interface{}) error
	DeleteField(key, namespace, name string) error
	GetValue(key string) (string, error)
	GetKeys(regex string) ([]string, error)
	SetKeyWithTTL(key string, value interface{}, duration time.Duration) error
}

func BuildFieldName(namespace, name string) string {
	return namespace + "/" + name
}

func BuildMixedTopKey(topKey, childKey string) string {
	return "{" + topKey + "}" + childKey
}

type RedisEx interface {
	GetFieldValue(key string, value interface{}) error
	Init(loadQueryRlsEventsFunc func(key, value interface{}) error) error
}
