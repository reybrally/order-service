package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
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

	MaxRetries int
	Backoff    time.Duration
}

type writerProducer struct {
	w   *kafka.Writer
	cfg ProducerConfig
}

func NewProducer(cfg ProducerConfig) (Producer, error) {
	if cfg.RequiredAcks == 0 {
		cfg.RequiredAcks = kafka.RequireAll
	}
	if cfg.BatchBytes == 0 {
		cfg.BatchBytes = 1 << 20
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 50 * time.Millisecond
	}
	if cfg.Compression == 0 {
		cfg.Compression = kafka.Snappy
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 5 * time.Second
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 200 * time.Millisecond
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}

	tr := &kafka.Transport{
		ClientID: cfg.ClientID,
	}

	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Balancer:               &kafka.Hash{},
		RequiredAcks:           cfg.RequiredAcks,
		BatchBytes:             int64(cfg.BatchBytes),
		BatchTimeout:           cfg.BatchTimeout,
		Compression:            cfg.Compression,
		Async:                  cfg.Async,
		AllowAutoTopicCreation: cfg.AllowAutoTopicCreation,
		Transport:              tr,
		WriteTimeout:           cfg.WriteTimeout,
		ReadTimeout:            cfg.WriteTimeout,
	}

	return &writerProducer{w: w, cfg: cfg}, nil
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

	var lastErr error
	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return lastErr
			}
			return err
		}

		writeCtx, cancel := context.WithTimeout(ctx, p.cfg.WriteTimeout)
		err := p.w.WriteMessages(writeCtx, msg)
		cancel()

		if err == nil {
			return nil
		}
		lastErr = err

		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return lastErr
		}

		time.Sleep(p.cfg.Backoff * time.Duration(attempt+1))
	}
	return lastErr
}

func (p *writerProducer) PublishJSON(ctx context.Context, topic string, key []byte, value any, headers map[string]string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if headers == nil {
		headers = map[string]string{"content-type": "application/json"}
	} else if _, ok := headers["content-type"]; !ok {
		headers["content-type"] = "application/json"
	}
	return p.Publish(ctx, topic, key, data, headers)
}

func (p *writerProducer) Close() error { return p.w.Close() }

func PublishEnvelope[T any](ctx context.Context, p Producer, topic string, key []byte, env Envelope[T], headers map[string]string) error {
	if env.OccurredAt.IsZero() {
		env.OccurredAt = time.Now().UTC()
	}
	if env.Meta.Producer == "" {
		env.Meta.Producer = "order-service"
	}
	return p.PublishJSON(ctx, topic, key, env, headers)
}
