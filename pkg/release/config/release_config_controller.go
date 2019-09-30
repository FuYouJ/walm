package config

import (
	"WarpCloud/walm/pkg/k8s"
	"WarpCloud/walm/pkg/kafka"
	errorModel "WarpCloud/walm/pkg/models/error"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	releaseModel "WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release"
	"WarpCloud/walm/pkg/release/utils"
	"encoding/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"reflect"
	"strings"
	"time"
	"transwarp/release-config/pkg/apis/transwarp/v1beta1"
	"k8s.io/api/core/v1"
	k8sutils "WarpCloud/walm/pkg/k8s/utils"
	"WarpCloud/walm/pkg/models/common"
)

// 动态依赖管理核心需求：
// 1. 保存release的依赖关系， 当被依赖的release的输出配置改变时， 依赖者可以自动更新。
// 2. 保存release的输出配置， 当安装release时可以注入依赖的输出配置。
// 3. 保存release的输入配置， 可以实时上报release 输入配置和输出配置到配置中心， 输入配置和输出配置要保持一致性
// 4. 用户可以获取release依赖关系， 输出配置， 输入配置， 当前release状态， 依赖这个release更新的状态。

const (
	defaultWorkers                       = 1
	defaultReloadDependingReleaseWorkers = 10
	defaultKafkaWorkers                  = 2

	defaultRetryReloadDelayTimeSecond = 5
)

type ReleaseConfigController struct {
	workingQueue                       workqueue.DelayingInterface
	workers                            int
	reloadDependingReleaseWorkingQueue workqueue.DelayingInterface
	reloadDependingReleaseWorkers      int
	kafkaWorkingQueue                  workqueue.DelayingInterface
	kafkaWorkers                       int
	k8sCache                           k8s.Cache
	releaseUseCase                     release.UseCase
	kafka                              kafka.Kafka
	retryReloadDelayTimeSecond         int64
}

func NewReleaseConfigController(k8sCache k8s.Cache, releaseUseCase release.UseCase, kafka kafka.Kafka, retryReloadDelayTimeSecond int64) *ReleaseConfigController {
	controller := &ReleaseConfigController{
		workingQueue:                       workqueue.NewNamedDelayingQueue("release-config"),
		workers:                            defaultWorkers,
		reloadDependingReleaseWorkingQueue: workqueue.NewNamedDelayingQueue("reload-depending-release"),
		reloadDependingReleaseWorkers:      defaultReloadDependingReleaseWorkers,
		kafkaWorkingQueue:                  workqueue.NewNamedDelayingQueue("kafka"),
		kafkaWorkers:                       defaultKafkaWorkers,
		k8sCache:                           k8sCache,
		releaseUseCase:                     releaseUseCase,
		kafka:                              kafka,
		retryReloadDelayTimeSecond:         retryReloadDelayTimeSecond,
	}

	if controller.retryReloadDelayTimeSecond == 0 {
		controller.retryReloadDelayTimeSecond = defaultRetryReloadDelayTimeSecond
	}

	return controller
}

func (controller *ReleaseConfigController) Start(stopChan <-chan struct{}) {
	defer func() {
		klog.Info("v2 release config controller stopped")
	}()
	klog.Info("v2 release config controller started")

	defer controller.workingQueue.ShutDown()
	for i := 0; i < controller.workers; i++ {
		go wait.Until(controller.worker, time.Second, stopChan)
	}

	defer controller.kafkaWorkingQueue.ShutDown()
	for i := 0; i < controller.kafkaWorkers; i++ {
		go wait.Until(controller.kafkaWorker, time.Second, stopChan)
	}

	defer controller.reloadDependingReleaseWorkingQueue.ShutDown()
	for i := 0; i < controller.reloadDependingReleaseWorkers; i++ {
		go wait.Until(controller.reloadDependingReleaseWorker, time.Second, stopChan)
	}

	controller.k8sCache.AddReleaseConfigHandler(controller.getReleaseConfigEventHandlerFuncs())
	controller.k8sCache.AddServiceHandler(controller.getServiceEventHandlerFuncs())
	<-stopChan
}

func (controller *ReleaseConfigController) getReleaseConfigEventHandlerFuncs()(
	AddFunc func(obj interface{}), UpdateFunc  func(old, cur interface{}), DeleteFunc func(obj interface{})){
	AddFunc = func(obj interface{}) {
		controller.enqueueReleaseConfig(obj)
		controller.enqueueKafka(obj)
	}
	UpdateFunc = func(old, cur interface{}) {
		oldReleaseConfig, ok := old.(*v1beta1.ReleaseConfig)
		if !ok {
			klog.Error("old object is not release config")
			return
		}
		curReleaseConfig, ok := cur.(*v1beta1.ReleaseConfig)
		if !ok {
			klog.Error("cur object is not release config")
			return
		}
		if needsEnqueueUpdatedReleaseConfig(oldReleaseConfig, curReleaseConfig) {
			controller.enqueueReleaseConfig(cur)
		}
		if !reflect.DeepEqual(oldReleaseConfig.Spec, curReleaseConfig.Spec) {
			controller.enqueueKafka(cur)
		}
	}
	DeleteFunc = func(obj interface{}) {
		controller.enqueueKafka(obj)
	}
	return
}

func (controller *ReleaseConfigController) getServiceEventHandlerFuncs()(
	AddFunc func(obj interface{}), UpdateFunc  func(old, cur interface{}), DeleteFunc func(obj interface{})){
	AddFunc = func(obj interface{}) {
		svc, ok := obj.(*v1.Service)
		if !ok {
			klog.Error("obj is not service")
			return
		}
		if k8sutils.IsDummyService(svc) {
			controller.enqueueV1Release(svc, controller.workingQueue)
			controller.enqueueV1Release(svc, controller.kafkaWorkingQueue)
		}
	}
	UpdateFunc = func(old, cur interface{}) {
		oldSvc, ok := old.(*v1.Service)
		if !ok {
			klog.Error("old object is not service")
			return
		}
		curSvc, ok := cur.(*v1.Service)
		if !ok {
			klog.Error("cur object is not service")
			return
		}
		if k8sutils.IsDummyService(oldSvc) && k8sutils.IsDummyService(curSvc) {
			oldDependencyMeta, err := k8sutils.GetDependencyMetaFromDummyServiceMetaStr(oldSvc.Annotations["transwarp.meta"])
			if err != nil {
				return
			}
			curDependencyMeta, err := k8sutils.GetDependencyMetaFromDummyServiceMetaStr(curSvc.Annotations["transwarp.meta"])
			if err != nil {
				return
			}
			if !reflect.DeepEqual(oldDependencyMeta, curDependencyMeta) {
				controller.enqueueV1Release(curSvc, controller.workingQueue)
				controller.enqueueV1Release(curSvc, controller.kafkaWorkingQueue)
			}
		}
	}
	DeleteFunc = func(obj interface{}) {
		svc, ok := obj.(*v1.Service)
		if !ok {
			klog.Error("obj is not service")
			return
		}
		if k8sutils.IsDummyService(svc) {
			controller.enqueueV1Release(svc, controller.kafkaWorkingQueue)
		}
	}
	return
}

func (controller *ReleaseConfigController) enqueueReleaseConfig(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Couldn't get key for object %#v: %v", obj, err)
		return
	}
	controller.workingQueue.Add(key)
}
// for compatible
func (controller *ReleaseConfigController) enqueueV1Release(dummySvc *v1.Service, queue workqueue.DelayingInterface) {
	releaseName := k8sutils.GetReleaseNameFromDummyService(dummySvc)
	if releaseName != "" {
		queue.Add(dummySvc.Namespace + "/" + releaseName)
	} else {
		klog.Warningf("can not get release name from dummy svc %s/%s", dummySvc.Namespace, dummySvc.Name)
	}
}

func (controller *ReleaseConfigController) enqueueKafka(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Couldn't get key for object %#v: %v", obj, err)
		return
	}
	controller.kafkaWorkingQueue.Add(key)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (controller *ReleaseConfigController) worker() {
	for {
		func() {
			key, quit := controller.workingQueue.Get()
			if quit {
				return
			}
			defer controller.workingQueue.Done(key)
			err := controller.syncReleaseConfig(key.(string))
			if err != nil {
				klog.Errorf("Error syncing release config: %v", err)
			}
		}()
	}
}

func (controller *ReleaseConfigController) kafkaWorker() {
	for {
		func() {
			key, quit := controller.kafkaWorkingQueue.Get()
			if quit {
				return
			}
			defer controller.kafkaWorkingQueue.Done(key)
			err := controller.publishToKafka(key.(string))
			if err != nil {
				klog.Errorf("failed to publish release config of %s to kafka: %s", key.(string), err.Error())
			}
		}()
	}
}

func (controller *ReleaseConfigController) publishToKafka(releaseKey string) error {
	klog.Infof("start to publish release config of %s to kafka", releaseKey)
	namespace, name, err := cache.SplitMetaNamespaceKey(releaseKey)
	if err != nil {
		return err
	}

	event := releaseModel.ReleaseConfigDeltaEvent{}

	var releaseVersion common.WalmVersion
	resourceFound := false
	resource, err := controller.k8sCache.GetResource(k8sModel.ReleaseConfigKind, namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			resource, err = controller.k8sCache.GetResource(k8sModel.InstanceKind, namespace, name)
			if err != nil {
				if errorModel.IsNotFoundError(err) {
					event.Type = releaseModel.Delete
					event.Data = &releaseModel.ReleaseConfigData{
						ReleaseConfig: k8sModel.ReleaseConfig{
							Meta: k8sModel.Meta{
								Namespace: namespace,
								Name:      name,
							},
						},
					}
				}else {
					klog.Errorf("failed to get instance of %s", releaseKey)
					return err
				}
			} else {
				resourceFound = true
				releaseVersion = common.WalmVersionV1
			}
		} else {
			klog.Errorf("failed to get release config of %s", releaseKey)
			return err
		}
	} else {
		resourceFound = true
		releaseVersion = common.WalmVersionV2
	}

	if resourceFound {
		release, err := controller.releaseUseCase.GetRelease(namespace, name)
		if err != nil {
			if errorModel.IsNotFoundError(err) {
				klog.Warningf("release %s is not found， ignore to publish release config to kafka", releaseKey)
				return nil
			}
			klog.Errorf("failed to get release %s : %s", releaseKey, err.Error())
			return err
		}
		event.Type = releaseModel.CreateOrUpdate
		event.Data = &releaseModel.ReleaseConfigData{
			ReleaseWalmVersion: releaseVersion,
		}
		if releaseVersion == common.WalmVersionV2 {
			event.Data.ReleaseConfig = *resource.(*k8sModel.ReleaseConfig)
		} else if releaseVersion == common.WalmVersionV1 {
			event.Data.ReleaseConfig = buildEventDataFromRelease(release)
		}
	}

	eventMsg, err := json.Marshal(event)
	if err != nil {
		klog.Errorf("failed to marshal event : %s", err.Error())
		return err
	}

	err = controller.kafka.SyncSendMessage(kafka.ReleaseConfigTopic, string(eventMsg))
	if err != nil {
		klog.Errorf("failed to send release config event of %s to kafka : %s", releaseKey, err.Error())
		return err
	}

	return nil
}

func buildEventDataFromRelease(release *releaseModel.ReleaseInfoV2) k8sModel.ReleaseConfig {
	return k8sModel.ReleaseConfig{
		Meta:                     k8sModel.NewMeta(k8sModel.ReleaseConfigKind, release.Namespace, release.Name, k8sModel.NewState("Ready", "", "")),
		Labels:                   release.ReleaseLabels,
		OutputConfig:             release.OutputConfigValues,
		ChartImage:               release.ChartImage,
		ChartName:                release.ChartName,
		ConfigValues:             release.ConfigValues,
		Dependencies:             release.Dependencies,
		ChartVersion:             release.ChartVersion,
		ChartAppVersion:          release.ChartAppVersion,
		Repo:                     release.RepoName,
		DependenciesConfigValues: release.DependenciesConfigValues,
	}
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (controller *ReleaseConfigController) reloadDependingReleaseWorker() {
	for {
		func() {
			key, quit := controller.reloadDependingReleaseWorkingQueue.Get()
			if quit {
				return
			}
			defer controller.reloadDependingReleaseWorkingQueue.Done(key)
			err := controller.reloadDependingRelease(key.(string))
			if err != nil {
				if strings.Contains(err.Error(), release.WaitReleaseTaskMsgPrefix) {
					klog.Warningf("depending release %s would be reloaded after %d second", key.(string), controller.retryReloadDelayTimeSecond)
					controller.reloadDependingReleaseWorkingQueue.AddAfter(key, time.Second*time.Duration(controller.retryReloadDelayTimeSecond))
				} else {
					klog.Errorf("Error reload depending release %s: %v", key.(string), err)
				}
			}
		}()
	}
}

func needsEnqueueUpdatedReleaseConfig(old *v1beta1.ReleaseConfig, cur *v1beta1.ReleaseConfig) bool {
	if utils.ConfigValuesDiff(old.Spec.OutputConfig, cur.Spec.OutputConfig) {
		return true
	}
	return false
}

func (controller *ReleaseConfigController) reloadDependingRelease(releaseKey string) error {
	klog.Infof("start to reload release %s", releaseKey)
	namespace, name, err := cache.SplitMetaNamespaceKey(releaseKey)
	if err != nil {
		return err
	}
	err = controller.releaseUseCase.ReloadRelease(namespace, name)
	if err != nil {
		klog.Errorf("failed to reload release %s/%s : %s", namespace, name, err.Error())
		return err
	}
	return nil
}

// 两级work queue设计初衷：利用work queue压缩相同key的功能， 尽可能地减少reload release的次数
// a 有多个依赖 b, c, d...， 当b, c, d... 同时更新了，a最好的情况是只更新一次
func (controller *ReleaseConfigController) syncReleaseConfig(releaseConfigKey string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(releaseConfigKey)
	if err != nil {
		return err
	}

	releaseConfigs, err := controller.k8sCache.ListReleaseConfigs("", "")
	if err != nil {
		klog.Errorf("failed to list all release configs : %s", err.Error())
		return err
	}
	for _, releaseConfig := range releaseConfigs {
		for _, dependedRelease := range releaseConfig.Dependencies {
			dependedReleaseNamespace, dependedReleaseName, err := utils.ParseDependedRelease(releaseConfig.Namespace, dependedRelease)
			if err != nil {
				continue
			}
			if dependedReleaseNamespace == namespace && dependedReleaseName == name {
				rc := &v1beta1.ReleaseConfig{}
				rc.Namespace = releaseConfig.Namespace
				rc.Name = releaseConfig.Name
				controller.enqueueDependingRelease(rc)
				break
			}
		}
	}

	return nil
}

func (controller *ReleaseConfigController) enqueueDependingRelease(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Couldn't get key for object %#v: %v", obj, err)
		return
	}
	controller.reloadDependingReleaseWorkingQueue.Add(key)
}
