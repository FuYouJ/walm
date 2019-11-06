package impl

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	walmRedis "WarpCloud/walm/pkg/redis"
	"WarpCloud/walm/pkg/setting"
	"encoding/json"
	"github.com/go-redis/redis"
	"k8s.io/klog"
	"time"
	"transwarp/cachex"
	"transwarp/cachex/rdscache"
)

type Redis struct {
	client *redis.Client
}

type RedisEx struct {
	storage cachex.Storage
	clientEx *cachex.Cachex
}

func (redisEx *RedisEx) GetFieldValue(key string) (interface{}, error) {
	value := k8s.EventList{}
	err := redisEx.clientEx.Get(key, &value)
	if err != nil {
		klog.Errorf("failed to get value of key %s from redisEx: %s", key, err.Error())
		return nil, err
	}
	return &value, nil
}

func (redisEx *RedisEx) Init(loadFunc func(key, value interface{}) error) error {
	 redisEx.clientEx = cachex.NewCachex(redisEx.storage, cachex.QueryFunc(loadFunc))
	 return nil
}

func (redis *Redis) GetFieldValue(key, namespace, name string) (value string, err error) {
	value, err = redis.client.HGet(key, walmRedis.BuildFieldName(namespace, name)).Result()
	if err != nil {
		if isKeyNotFoundError(err) {
			klog.Warningf("field %s/%s of key %s is not found in redis", namespace, name, key)
			err = errorModel.NotFoundError{}
			return
		}
		klog.Errorf("failed to get field %s/%s of key %s from redis: %s", namespace, name, key, err.Error())
		return
	}
	return
}

func (redis *Redis) GetFieldValues(key, namespace, filter string) (values []string, err error) {
	values = []string{}
	if namespace == "" && filter == ""{
		releaseCacheMap, err := redis.client.HGetAll(key).Result()
		if err != nil {
			klog.Errorf("failed to get all the fields of key %s from redis: %s", key, err.Error())
			return nil, err
		}
		for _, releaseCacheStr := range releaseCacheMap {
			values = append(values, releaseCacheStr)
		}
	} else {
		filter := buildHScanFilter(namespace, filter)
		// ridiculous logic: scan result contains both key and value
		scanResult, _, err := redis.client.HScan(key, 0, filter, 10000).Result()
		if err != nil {
			klog.Errorf("failed to scan the redis with filter=%s : %s", filter, err.Error())
			return nil, err
		}

		for i := 1; i < len(scanResult); i += 2 {
			values = append(values, scanResult[i])
		}
	}
	return
}

func (redis *Redis) GetFieldValuesByNames(key string, fieldNames ...string) (values []string, err error) {
	objects, err := redis.client.HMGet(key, fieldNames...).Result()
	if err != nil {
		klog.Errorf("failed to get fields %v of key %s from redis : %s", fieldNames, key, err.Error())
		return nil, err
	}
	values = []string{}
	for _, object := range objects {
		if object != nil {
			values = append(values, object.(string))
		}
	}
	return
}

func (redis *Redis) SetFieldValues(key string, fieldValues map[string]interface{}) error {
	if len(fieldValues) == 0 {
		return nil
	}
	marshaledFieldValues := map[string]interface{}{}
	for k, value := range fieldValues {
		valueStr, err := json.Marshal(value)
		if err != nil {
			klog.Errorf("failed to marshal value : %s", err.Error())
			return err
		}
		marshaledFieldValues[k] = string(valueStr)
	}
	_, err := redis.client.HMSet(key, marshaledFieldValues).Result()
	if err != nil {
		klog.Errorf("failed to set to redis : %s", err.Error())
		return err
	}
	return nil
}

func (redis *Redis) DeleteField(key, namespace, name string) error {
	_, err := redis.client.HDel(key, walmRedis.BuildFieldName(namespace, name)).Result()
	if err != nil {
		klog.Errorf("failed to delete filed %s/%s of key %s from redis: %s", namespace, name, key, err.Error())
		return err
	}
	return nil
}

func NewRedisClient(redisConfig *setting.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         redisConfig.Addr,
		Password:     redisConfig.Password,
		DB:           redisConfig.DB,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	})
}

func NewRedis(redisClient *redis.Client) *Redis {
	return &Redis{
		client: redisClient,
	}
}

func NewRedisEx(config *setting.RedisConfig, ttl time.Duration) *RedisEx {

	storage := rdscache.NewRdsCache("tcp", config.Addr, rdscache.PoolConfig{DB: config.DB, Password: config.Password}, rdscache.RdsKeyPrefixOption("events"), rdscache.RdsDefaultTTLOption(ttl))
	return &RedisEx{storage: storage}
}
