package kafka

import "time"

type Envelope[T any] struct {
	EventType  string    `json:"event_type"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
	EntityID   string    `json:"entity_id"`
	Payload    T         `json:"payload"`
	Meta       Meta      `json:"meta"`
}

type Meta struct {
	Producer string `json:"producer"`
	TraceID  string `json:"trace_id"`
	Source   string `json:"source"`
}
