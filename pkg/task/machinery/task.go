package machinery

import (
	errorModel "WarpCloud/walm/pkg/models/error"
	taskModel "WarpCloud/walm/pkg/models/task"
	"WarpCloud/walm/pkg/setting"
	"WarpCloud/walm/pkg/task"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/backends/result"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	"k8s.io/klog"
	"os"
	"time"
	"strings"
	"fmt"
	brokerredis "github.com/RichardKnop/machinery/v1/brokers/redis"
	backendredis "github.com/RichardKnop/machinery/v1/backends/redis"
	"github.com/RichardKnop/machinery/v1/common"
	"github.com/RichardKnop/machinery/v1/log"
)

type Task struct {
	server *machinery.Server
	worker *machinery.Worker
}

func (task *Task) GetTaskState(sig *taskModel.TaskSig) (state task.TaskState, err error) {
	taskSig := convertTaskSig(sig)
	if taskSig == nil {
		return nil, errorModel.NotFoundError{}
	}
	asyncResult := result.NewAsyncResult(taskSig, task.server.GetBackend())
	taskState := asyncResult.GetState()
	if taskState == nil || taskState.TaskName == "" {
		return nil, errorModel.NotFoundError{}
	}
	state = &TaskStateAdaptor{
		taskState:      taskState,
		taskTimeoutSec: sig.TimeoutSec,
	}
	return
}

func convertTaskSig(sig *taskModel.TaskSig) *tasks.Signature {
	if sig == nil || sig.UUID == "" {
		return nil
	}
	return &tasks.Signature{
		Name: sig.Name,
		UUID: sig.UUID,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: sig.Arg,
			},
		},
	}
}

func (task *Task) RegisterTask(taskName string, taskRunner func(taskArgs string) error) error {
	err := task.server.RegisterTask(taskName, taskRunner)
	if err != nil {
		klog.Errorf("failed to register task %s : %s", taskName, err.Error())
		return err
	}
	return nil
}

func (task *Task) SendTask(taskName, taskArgs string, timeoutSec int64) (*taskModel.TaskSig, error) {
	taskSig := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: taskArgs,
			},
		},
	}
	_, err := task.server.SendTask(taskSig)
	if err != nil {
		klog.Errorf("failed to send %s : %s", taskName, err.Error())
		return nil, err
	}

	sig := &taskModel.TaskSig{
		Name:       taskName,
		UUID:       taskSig.UUID,
		Arg:        taskArgs,
		TimeoutSec: timeoutSec,
	}
	return sig, nil
}

func (task *Task) TouchTask(sig *taskModel.TaskSig, pollingIntervalSec int64) (error) {
	taskSig := convertTaskSig(sig)
	if taskSig == nil {
		return errorModel.NotFoundError{}
	}
	asyncResult := result.NewAsyncResult(taskSig, task.server.GetBackend())
	_, err := asyncResult.GetWithTimeout(time.Duration(sig.TimeoutSec)*time.Second, time.Duration(pollingIntervalSec)*time.Second)
	if err != nil {
		klog.Errorf("touch task %s-%s failed: %s", sig.Name, sig.UUID, err.Error())
		return err
	}
	return nil
}

func (task *Task) PurgeTaskState(sig *taskModel.TaskSig) (error) {
	if sig == nil || sig.UUID == "" {
		return nil
	}
	err := task.server.GetBackend().PurgeState(sig.UUID)
	if err != nil {
		klog.Errorf("failed to purge task state : %s", err.Error())
		return err
	}
	return nil
}

func (task *Task) StartWorker() {
	task.worker = task.server.NewWorker(os.Getenv("Pod_Name"), 100)
	errorsChan := make(chan error)
	task.worker.LaunchAsync(errorsChan)
	go func(errChan chan error) {
		if err := <-errChan; err != nil {
			klog.Error(err.Error())
		}
	}(errorsChan)
	klog.Info("worker starting to consume tasks")
}

func (task *Task) StopWorker(timeoutSec int64) {
	quitChan := make(chan struct{})
	go func() {
		task.worker.Quit()
		close(quitChan)
	}()
	select {
	case <-quitChan:
		klog.Info("worker stopped consuming tasks successfully")
	case <-time.After(time.Second * time.Duration(timeoutSec)):
		klog.Warning("worker stopped consuming tasks failed after 30 seconds")
	}
}

func NewTask(c *setting.TaskConfig) (*Task, error) {
	taskConfig := &config.Config{
		Broker:          c.Broker,
		DefaultQueue:    c.DefaultQueue,
		ResultBackend:   c.ResultBackend,
		ResultsExpireIn: c.ResultsExpireIn,
		NoUnixSignals:   true,
		Redis: &config.RedisConfig{
			MaxIdle:                3,
			IdleTimeout:            240,
			ReadTimeout:            15,
			WriteTimeout:           15,
			ConnectTimeout:         15,
			DelayedTasksPollPeriod: 20,
		},
	}

	redisOptions := common.OtherGoRedisOptions{
		MaxRetries: 15,
	}

	if c.RedisConfig != nil {
		if c.RedisConfig.MaxRetries > 0 {
			redisOptions.MaxRetries = c.RedisConfig.MaxRetries
		}
		redisOptions.MinRetryBackoff = time.Millisecond * time.Duration(c.RedisConfig.MinRetryBackoff)
		redisOptions.MaxRetryBackoff = time.Millisecond * time.Duration(c.RedisConfig.MaxRetryBackoff)
	}

	brokerRedisAddrs, err := buildRedisAddrs(taskConfig.Broker)
	if err != nil {
		return nil, err
	}

	fibonacci := func() func() int {
		a, b := 0, 1
		return func() int {
			a, b = b, a+b
			return a
		}
	}

	retryFunc := func() func(chan int) {
		retryIn := 0
		fibonacci := fibonacci()
		return func(stopChan chan int) {
			if retryIn > 0 {
				durationString := fmt.Sprintf("%vs", retryIn)
				duration, _ := time.ParseDuration(durationString)

				log.WARNING.Printf("Retrying in %v seconds", retryIn)

				select {
				case <-stopChan:
					break
				case <-time.After(duration):
					break
				}
			}
			if retryIn < 30 {
				retryIn = fibonacci()
			}
			if retryIn > 30 {
				retryIn = 30
			}
		}
	}
	brokerServer, err := brokerredis.NewGREx(taskConfig, brokerRedisAddrs, &redisOptions, retryFunc())
	if err != nil {
		klog.Errorf("failed to new redis broker server : %s", err.Error())
		return nil, err
	}

	backendRedisAddrs, err := buildRedisAddrs(taskConfig.ResultBackend)
	if err != nil {
		return nil, err
	}
	backendServer, err := backendredis.NewGREx(taskConfig, backendRedisAddrs, &redisOptions)
	if err != nil {
		klog.Errorf("failed to new backend redis server : %s", err.Error())
		return nil, err
	}

	server := machinery.NewServerWithBrokerBackend(taskConfig, brokerServer, backendServer)
	return &Task{
		server: server,
	}, nil
}

func buildRedisAddrs(connStr string) ([]string, error) {
	if strings.HasPrefix(connStr, "redis://") {
		return strings.Split(connStr, ","), nil
	} else {
		return nil, fmt.Errorf("redis connection string should be prefixed with redis://, instead got %s", connStr)
	}
}
