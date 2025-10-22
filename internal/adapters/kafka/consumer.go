// internal/adapters/kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	kgo "github.com/segmentio/kafka-go"
)

// Message — обертка над kafka-go с уже распарсенным Envelope.
type Message struct {
	Topic   string
	Key     []byte
	Headers map[string]string
	// Сырые байты, если нужно логировать/слать в DLQ
	Raw kgo.Message
	// Декодированный конверт с RawPayload (распарсим payload позднее по event_type)
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
	MinBytes          int           // 1<<10
	MaxBytes          int           // 10<<20
	MaxWait           time.Duration // 100 * time.Millisecond
	SessionTimeout    time.Duration // 10 * time.Second
	RebalanceTimeout  time.Duration // 10 * time.Second
	HeartbeatInterval time.Duration // 3 * time.Second
	StartOffset       int64         // kgo.FirstOffset / kgo.LastOffset
	// Ретраи обработки
	MaxRetries int           // 5
	Backoff    time.Duration // 200 * time.Millisecond
}

type readerConsumer struct {
	cfg    ConsumerConfig
	reader *kgo.Reader // создаём per-topic в Subscribe
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
			// Контекст закрыт — graceful shutdown
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			// Временная ошибка брокера — подождём и продолжим
			time.Sleep(200 * time.Millisecond)
			continue
		}

		msg := toMessage(topic, m)

		// Ретраим обработчик (at-least-once)
		var hErr error
		for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
			hErr = handler(ctx, msg)
			if hErr == nil {
				break
			}
			// если контекст умер — выходим тихо
			if ctx.Err() != nil {
				return nil
			}
			time.Sleep(c.cfg.Backoff * time.Duration(attempt+1))
		}

		if hErr != nil {
			// сюда можно добавить отправку в DLQ через твой Producer
			// _ = dlqProducer.Publish(...)
			// но для старта просто пропустим и закоммитим, чтобы не застрять
		}

		// Коммитим независимо (at-least-once семантика + идемпотентные обработчики)
		if err := r.CommitMessages(ctx, m); err != nil {
			// ошибка коммита — попробуем ещё раз на следующей итерации
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
	_ = json.Unmarshal(m.Value, &env) // намеренно игнорим ошибку здесь — handler может перепарсить сам
	return Message{
		Topic:    topic,
		Key:      m.Key,
		Headers:  hdrs,
		Raw:      m,
		Envelope: env,
	}
}
