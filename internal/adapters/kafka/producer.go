package kafka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"time"
)

type Producer interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error

	PublishJSON(ctx context.Context, topic string, key []byte, value any, headers map[string]string) error

	Close() error
}

type ProducerConfig struct {
	Brokers                []string
	ClientID               string
	RequiredAcks           kafka.RequiredAcks
	BatchBytes             int
	BatchTimeout           time.Duration
	Compression            kafka.Compression
	Async                  bool
	WriteTimeout           time.Duration
	AllowAutoTopicCreation bool
}

type writerProducer struct {
	w *kafka.Writer
}

func NewProducer(cfg ProducerConfig) (Producer, error) {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Balancer:               &kafka.Hash{},
		RequiredAcks:           cfg.RequiredAcks,
		BatchBytes:             int64(cfg.BatchBytes),
		BatchTimeout:           cfg.BatchTimeout,
		Compression:            cfg.Compression,
		Async:                  cfg.Async,
		AllowAutoTopicCreation: cfg.AllowAutoTopicCreation,
	}
	return &writerProducer{w: w}, nil
}

func (p *writerProducer) Publish(ctx context.Context, topic string, key, value []byte, headers map[string]string) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
		Time:  time.Now().UTC(),
	}
	if len(headers) > 0 {
		msg.Headers = make([]kafka.Header, 0, len(headers))
		for k, v := range headers {
			msg.Headers = append(msg.Headers, kafka.Header{Key: k, Value: []byte(v)})
		}
	}

	return p.w.WriteMessages(ctx, msg)
}

func (p *writerProducer) PublishJSON(ctx context.Context, topic string, key []byte, value any, headers map[string]string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return p.Publish(ctx, topic, key, data, headers)
}

func (p *writerProducer) Close() error { return p.w.Close() }
