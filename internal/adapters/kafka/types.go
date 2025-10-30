package kafka

type OrderUpserted struct {
	OrderUID string `json:"order_uid"`
}

type OrderDeleted struct {
	OrderUID string `json:"order_uid"`
}
