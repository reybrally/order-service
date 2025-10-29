package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	kgo "github.com/segmentio/kafka-go"
)

type Message struct {
	Topic    string
	Key      []byte
	Headers  map[string]string
	Raw      kgo.Message
	Envelope Envelope[json.RawMessage]
}

type Handler func(ctx context.Context, msg Message) error

type Consumer interface {
	Subscribe(ctx context.Context, topic string, groupID string, handler Handler) error
	Close() error
}

type ConsumerConfig struct {
	Brokers           []string
	ClientID          string
	MinBytes          int
	MaxBytes          int
	MaxWait           time.Duration
	SessionTimeout    time.Duration
	RebalanceTimeout  time.Duration
	HeartbeatInterval time.Duration
	StartOffset       int64
	MaxRetries        int
	Backoff           time.Duration
}

type readerConsumer struct {
	cfg    ConsumerConfig
	reader *kgo.Reader
}

func NewConsumer(cfg ConsumerConfig) Consumer {
	return &readerConsumer{cfg: cfg}
}

func (c *readerConsumer) Subscribe(ctx context.Context, topic string, groupID string, handler Handler) error {
	r := kgo.NewReader(kgo.ReaderConfig{
		Brokers:           c.cfg.Brokers,
		GroupID:           groupID,
		Topic:             topic,
		MinBytes:          c.cfg.MinBytes,
		MaxBytes:          c.cfg.MaxBytes,
		MaxWait:           c.cfg.MaxWait,
		StartOffset:       c.cfg.StartOffset,
		SessionTimeout:    c.cfg.SessionTimeout,
		RebalanceTimeout:  c.cfg.RebalanceTimeout,
		HeartbeatInterval: c.cfg.HeartbeatInterval,
	})
	c.reader = r
	defer r.Close()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}

		msg := toMessage(topic, m)

		var hErr error
		for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
			hErr = handler(ctx, msg)
			if hErr == nil {
				break
			}
			if ctx.Err() != nil {
				return nil
			}
			time.Sleep(c.cfg.Backoff * time.Duration(attempt+1))
		}

		if hErr != nil {
			// TODO : implement
		}

		if err := r.CommitMessages(ctx, m); err != nil {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (c *readerConsumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

func toMessage(topic string, m kgo.Message) Message {
	hdrs := make(map[string]string, len(m.Headers))
	for _, h := range m.Headers {
		hdrs[h.Key] = string(h.Value)
	}
	var env Envelope[json.RawMessage]
	_ = json.Unmarshal(m.Value, &env)
	return Message{
		Topic:    topic,
		Key:      m.Key,
		Headers:  hdrs,
		Raw:      m,
		Envelope: env,
	}
}
