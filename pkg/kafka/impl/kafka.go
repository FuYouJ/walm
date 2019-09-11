package impl

import (
	"WarpCloud/walm/pkg/setting"
	"crypto/tls"
	"crypto/x509"
	"github.com/Shopify/sarama"
	"io/ioutil"
	"k8s.io/klog"
)

type Kafka struct {
	setting.KafkaConfig
	syncProducer sarama.SyncProducer
}

func (kafkaImpl *Kafka) SyncSendMessage(topic, message string) error {
	if !kafkaImpl.Enable {
		klog.Warningf("kafka client is not enabled, failed to send message %s to topic %s", message, topic)
		return nil
	}
	_, _, err := kafkaImpl.syncProducer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	})

	if err != nil {
		klog.Errorf("failed to send msg %s to topic %s : %s", message, topic, err.Error())
		return err
	}

	klog.Infof("succeed to send msg %s to topic %s", message, topic)
	return nil
}

func NewKafka(kafkaConfig *setting.KafkaConfig) (*Kafka, error) {
	if kafkaConfig == nil {
		kafkaConfig = &setting.KafkaConfig{}
	}

	kafkaClient := &Kafka{
		KafkaConfig: *setting.Config.KafkaConfig,
	}

	if kafkaClient.Enable {
		config := sarama.NewConfig()
		config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
		config.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
		config.Producer.Return.Successes = true
		tlsConfig := createTlsConfiguration(kafkaConfig)
		if tlsConfig != nil {
			config.Net.TLS.Config = tlsConfig
			config.Net.TLS.Enable = true
		}

		// On the broker side, you may want to change the following settings to get
		// stronger consistency guarantees:
		// - For your broker, set `unclean.leader.election.enable` to false
		// - For the topic, you could increase `min.insync.replicas`.

		syncProducer, err := sarama.NewSyncProducer(kafkaClient.Brokers, config)
		if err != nil {
			klog.Errorf("Failed to start Sarama producer: %s", err.Error())
			return nil, err
		}
		kafkaClient.syncProducer = syncProducer
	} else {
		klog.Warning("kafka client is not enabled")
	}
	return kafkaClient, nil
}

func createTlsConfiguration(kafkaConfig *setting.KafkaConfig) (t *tls.Config) {
	if kafkaConfig.CertFile != "" && kafkaConfig.KeyFile != "" && kafkaConfig.CaFile != "" {
		cert, err := tls.LoadX509KeyPair(kafkaConfig.CertFile, kafkaConfig.KeyFile)
		if err != nil {
			klog.Fatal(err)
		}

		caCert, err := ioutil.ReadFile(kafkaConfig.CaFile)
		if err != nil {
			klog.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: kafkaConfig.VerifySsl,
		}
	}
	// will be nil by default if nothing is provided
	return t
}
