package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	PerMessageTimeout time.Duration

	DLQTopic    string
	DLQProducer Producer
}

type readerConsumer struct {
	cfg    ConsumerConfig
	reader *kgo.Reader
}

func NewConsumer(cfg ConsumerConfig) Consumer {
	if cfg.PerMessageTimeout <= 0 {
		cfg.PerMessageTimeout = 10 * time.Second
	}
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

		if msg.Envelope.EventType == "" {
			_ = c.pushDLQ(ctx, msg, fmt.Errorf("empty event_type"))
			_ = r.CommitMessages(ctx, m)
			continue
		}
		if msg.Envelope.Version <= 0 {
			_ = c.pushDLQ(ctx, msg, fmt.Errorf("invalid version: %d", msg.Envelope.Version))
			_ = r.CommitMessages(ctx, m)
			continue
		}

		var hErr error
		for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
			if ctx.Err() != nil {
				return nil
			}
			msgCtx, cancel := context.WithTimeout(ctx, c.cfg.PerMessageTimeout)
			hErr = safeHandle(msgCtx, handler, msg)
			cancel()

			if hErr == nil {
				break
			}
			time.Sleep(c.cfg.Backoff * time.Duration(attempt+1))
		}

		if hErr != nil {
			_ = c.pushDLQ(ctx, msg, hErr)
			continue
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

func safeHandle(ctx context.Context, h Handler, msg Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return h(ctx, msg)
}

func (c *readerConsumer) pushDLQ(ctx context.Context, msg Message, cause error) error {
	if c.cfg.DLQProducer == nil || c.cfg.DLQTopic == "" {
		return nil
	}
	payload := struct {
		FailedAt    time.Time                 `json:"failed_at"`
		Error       string                    `json:"error"`
		Topic       string                    `json:"topic"`
		KeyBase64   string                    `json:"key_base64,omitempty"`
		ValueBase64 string                    `json:"value_base64"`
		Envelope    Envelope[json.RawMessage] `json:"envelope"`
		Partition   int                       `json:"partition"`
		Offset      int64                     `json:"offset"`
		Time        time.Time                 `json:"time"`
		Headers     map[string]string         `json:"headers,omitempty"`
	}{
		FailedAt:    time.Now().UTC(),
		Error:       cause.Error(),
		Topic:       msg.Topic,
		KeyBase64:   b64(msg.Raw.Key),
		ValueBase64: b64(msg.Raw.Value),
		Envelope:    msg.Envelope,
		Partition:   msg.Raw.Partition,
		Offset:      msg.Raw.Offset,
		Time:        msg.Raw.Time,
		Headers:     msg.Headers,
	}
	return c.cfg.DLQProducer.PublishJSON(ctx, c.cfg.DLQTopic, msg.Key, payload, nil)
}

func b64(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return jsonBase64(b)
}

func jsonBase64(b []byte) string {
	enc, _ := json.Marshal(b)
	if len(enc) >= 2 && enc[0] == '"' && enc[len(enc)-1] == '"' {
		return string(enc[1 : len(enc)-1])
	}
	return string(enc)
}
