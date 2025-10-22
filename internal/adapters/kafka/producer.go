package kafka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"time"
)

type Producer interface {
	// Publish отправляет уже готовые bytes (если сериализуешь сам)
	Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error

	// PublishJSON сериализует value в JSON и отправляет
	PublishJSON(ctx context.Context, topic string, key []byte, value any, headers map[string]string) error

	Close() error
}

type ProducerConfig struct {
	Brokers                []string           // ["localhost:9092"]
	ClientID               string             // "order-service"
	RequiredAcks           kafka.RequiredAcks // kafka.RequireAll для acks=all
	BatchBytes             int                // напр. 1048576 (1MB)
	BatchTimeout           time.Duration      // напр. 50 * time.Millisecond
	Compression            kafka.Compression  // kafka.Snappy или Lz4/None
	Async                  bool               // оставь false для синхронной отправки
	WriteTimeout           time.Duration      // 5s
	AllowAutoTopicCreation bool               // true локально, false в проде
}

type writerProducer struct {
	w *kafka.Writer
}

func NewProducer(cfg ProducerConfig) (Producer, error) {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{}, // важен порядок по ключу
		RequiredAcks: cfg.RequiredAcks,
		BatchBytes:   int64(cfg.BatchBytes),
		BatchTimeout: cfg.BatchTimeout,
		Compression:  cfg.Compression,
		Async:        cfg.Async,
		// Note: Topic не задаём здесь, чтобы писать в разные топики
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

	// Доп. safety: honor таймаут из cfg, если в ctx его нет — можно оборачивать с timeout выше по стеку.
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
