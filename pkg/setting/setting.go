package setting

import (
	"github.com/ghodss/yaml"
	"io/ioutil"
	"k8s.io/klog"
)

var Config WalmConfig

type HttpConfig struct {
	HTTPPort int    `json:"port,default=9999"`
	TLS      bool   `json:"tls"`
	TlsKey   string `json:"tlsKey"`
	TlsCert  string `json:"tlsCert"`
}

type ChartRepo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type KubeConfig struct {
	Config  string `json:"config"`
	Context string `json:"context"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	// Maximum number of retries before giving up.
	// Default is 15.
	MaxRetries int `json:"maxRetries"`
	// Minimum backoff between each retry.
	// Default is 8 milliseconds; -1 disables backoff.
	MinRetryBackoff int64 `json:"minRetryBackoff"`
	// Maximum backoff between each retry.
	// Default is 512 milliseconds; -1 disables backoff.
	MaxRetryBackoff int64 `json:"maxRetryBackoff"`
}

type KafkaConfig struct {
	Enable    bool     `json:"enable"`
	Brokers   []string `json:"brokers"`
	CertFile  string   `json:"certFile"`
	KeyFile   string   `json:"keyFile"`
	CaFile    string   `json:"caFile"`
	VerifySsl bool     `json:"verifySsl"`
}

type TaskConfig struct {
	Broker          string       `json:"broker"`
	DefaultQueue    string       `json:"default_queue"`
	ResultBackend   string       `json:"result_backend"`
	ResultsExpireIn int          `json:"results_expire_in"`
	RedisConfig     *RedisConfig `json:"redisConfig"`
}

type AdditionAppConfig struct {
	TosVersion  string                 `json:"tosVersion"`
	ExtraConfig map[string]interface{} `json:"extraConfig"`
}

type WalmConfig struct {
	Debug             bool               `json:"debug"`
	LogConfig         *LogConfig         `json:"logConfig"`
	HttpConfig        *HttpConfig        `json:"serverConfig"`
	RepoList          []*ChartRepo       `json:"repoList"`
	KubeConfig        *KubeConfig        `json:"kubeConfig"`
	RedisConfig       *RedisConfig       `json:"redisConfig"`
	KafkaConfig       *KafkaConfig       `json:"kafkaConfig"`
	TaskConfig        *TaskConfig        `json:"taskConfig"`
	JsonnetConfig     *JsonnetConfig     `json:"jsonnetConfig"`
	ChartImageConfig  *ChartImageConfig  `json:"chartImageConfig"`
	CrdConfig         *CrdConfig         `json:"crdConfig"`
	ElectorConfig     *ElectorConfig     `json:"electorConfig"`
	AdditionAppConfig *AdditionAppConfig `json:"additionAppConfig"`
	//only for test
	ChartImageRegistry string `json:"chartImageRegistry"`
}

type CrdConfig struct {
	NotNeedInstance    bool `json:"notNeedInstance"`
	EnableMigrationCRD bool `json:"enableMigrationCRD"`
}

type ElectorConfig struct {
	LockNamespace string `json:"lockNamespace" description:"pod namespace"`
	LockIdentity  string `json:"lockIdentity" description:"pod name"`
	ElectionId    string `json:"electionId" description:"election id"`
}

type ChartImageConfig struct {
	CacheRootDir string `json:"cacheRootDir"`
}

type LogConfig struct {
	Level  string `json:"level"`
	LogDir string `json:"logDir"`
}

type JsonnetConfig struct {
	CommonTemplateFilesPath string `json:"commonTemplateFilesPath"`
}

// StartResyncReleaseCaches sets values from the environment.
func InitConfig(configPath string) {
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		klog.Fatalf("Read config file faild! %s\n", err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		klog.Fatalf("Unmarshal config file faild! %s\n", err.Error())
	}
	if Config.AdditionAppConfig == nil {
		Config.AdditionAppConfig = &AdditionAppConfig{
			TosVersion: "1.9",
		}
	}
	if Config.AdditionAppConfig.TosVersion == "" {
		Config.AdditionAppConfig.TosVersion = "1.9"
	}
}

func InitDummyConfig() {
	if Config.AdditionAppConfig == nil {
		Config.AdditionAppConfig = &AdditionAppConfig{
			TosVersion: "1.9",
		}
	}
	if Config.AdditionAppConfig.TosVersion == "" {
		Config.AdditionAppConfig.TosVersion = "1.9"
	}
}
