package kafka

import "time"

type Envelope[T any] struct {
	EventType  string    `json:"event_type"`  // "order.upserted"
	Version    int       `json:"version"`     // 1
	OccurredAt time.Time `json:"occurred_at"` // UTC
	EntityID   string    `json:"entity_id"`   // обычно = order_uid (дублируем key)
	Payload    T         `json:"payload"`     // полезная нагрузка
	Meta       Meta      `json:"meta"`
}

type Meta struct {
	Producer string `json:"producer"` // "order-service"
	TraceID  string `json:"trace_id"` // прокидывай из контекста
	Source   string `json:"source"`   // "http-api" | "seeder" | ...
}
