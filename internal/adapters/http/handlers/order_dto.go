package handlers

import (
	"errors"
	"time"

	"github.com/reybrally/order-service/internal/domain/order"
)

type OrderUpsertRequest struct {
	OrderUID        *string `json:"order_uid,omitempty"`
	TrackNumber     string  `json:"track_number"`
	Entry           string  `json:"entry"`
	Locale          string  `json:"locale"`
	CustomerID      string  `json:"customer_id"`
	DeliveryService string  `json:"delivery_service"`
	ShardKey        string  `json:"shard_key"`
	SmID            int64   `json:"sm_id"`
	OofShard        int64   `json:"oof_shard"`

	Delivery DeliveryDTO `json:"delivery"`
	Payment  PaymentDTO  `json:"payment"`
	Items    []ItemDTO   `json:"items"`
}

type DeliveryDTO struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type PaymentDTO struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int64  `json:"amount"`
	PaymentDtRFC string `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int64  `json:"delivery_cost"`
	GoodsTotal   int64  `json:"goods_total"`
	CustomFee    int64  `json:"custom_fee"`
}

type ItemDTO struct {
	ChrtID      string `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int64  `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"item_name"`
	Sale        int64  `json:"sale"`
	Size        int64  `json:"item_size"`
	TotalPrice  int64  `json:"total_price"`
	NmID        string `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int64  `json:"status"`
}

func (r OrderUpsertRequest) ToModel() (order.Order, error) {
	// Лёгкая синтаксическая проверка «обязательных» (бизнес-валидация — в твоём validation пакете)
	if r.TrackNumber == "" || r.Entry == "" || r.Locale == "" || r.CustomerID == "" || r.DeliveryService == "" {
		return order.Order{}, errors.New("missing required fields")
	}
	// Парсим время оплаты (клиент присылает строкой RFC3339)
	var payTime time.Time
	if r.Payment.PaymentDtRFC != "" {
		t, err := time.Parse(time.RFC3339, r.Payment.PaymentDtRFC)
		if err != nil {
			return order.Order{}, errors.New("invalid payment_dt format (want RFC3339)")
		}
		payTime = t
	}

	out := order.Order{
		OrderUID:          derefStr(r.OrderUID),
		TrackNumber:       r.TrackNumber,
		Entry:             r.Entry,
		Locale:            r.Locale,
		InternalSignature: "",
		CustomerId:        r.CustomerID,
		DeliveryService:   r.DeliveryService,
		ShardKey:          r.ShardKey,
		SmId:              r.SmID,
		OofShard:          r.OofShard,

		Delivery: order.Delivery{
			Name:    r.Delivery.Name,
			Phone:   r.Delivery.Phone,
			Zip:     r.Delivery.Zip,
			City:    r.Delivery.City,
			Address: r.Delivery.Address,
			Region:  r.Delivery.Region,
			Email:   r.Delivery.Email,
		},
		Payment: order.Payment{
			Transaction:  r.Payment.Transaction,
			RequestId:    r.Payment.RequestID,
			Currency:     r.Payment.Currency,
			Provider:     r.Payment.Provider,
			Amount:       r.Payment.Amount,
			PaymentDt:    payTime,
			Bank:         r.Payment.Bank,
			DeliveryCost: r.Payment.DeliveryCost,
			GoodsTotal:   r.Payment.GoodsTotal,
			CustomFee:    r.Payment.CustomFee,
		},
		Items: make([]order.Item, 0, len(r.Items)),
	}

	for _, it := range r.Items {
		out.Items = append(out.Items, order.Item{
			ChrtId:      it.ChrtID,
			TrackNumber: it.TrackNumber,
			Price:       it.Price,
			Rid:         it.Rid,
			Name:        it.Name,
			Sale:        it.Sale,
			Size:        it.Size,
			TotalPrice:  it.TotalPrice,
			NmId:        it.NmID,
			Brand:       it.Brand,
			Status:      it.Status,
		})
	}

	return out, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// OrderResponse — DTO для ответа клиенту
type OrderResponse struct {
	OrderUID        string           `json:"order_uid"`
	TrackNumber     string           `json:"track_number"`
	Entry           string           `json:"entry"`
	Locale          string           `json:"locale"`
	CustomerID      string           `json:"customer_id"`
	DeliveryService string           `json:"delivery_service"`
	DateCreated     string           `json:"date_created"`
	Delivery        DeliveryResponse `json:"delivery"`
	Payment         PaymentResponse  `json:"payment"`
	Items           []ItemResponse   `json:"items"`
}

// DeliveryResponse — вложенная структура для ответа
type DeliveryResponse struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

// PaymentResponse — платежная часть
type PaymentResponse struct {
	Transaction  string `json:"transaction"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int64  `json:"amount"`
	PaymentDt    string `json:"payment_dt"` // RFC3339
	Bank         string `json:"bank"`
	DeliveryCost int64  `json:"delivery_cost"`
	GoodsTotal   int64  `json:"goods_total"`
	CustomFee    int64  `json:"custom_fee"`
}

// ItemResponse — элементы заказа
type ItemResponse struct {
	ChrtID      string `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int64  `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"item_name"`
	Sale        int64  `json:"sale"`
	Size        int64  `json:"item_size"`
	TotalPrice  int64  `json:"total_price"`
	NmID        string `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int64  `json:"status"`
}

// ToResponse — маппинг domain.Order → OrderResponse
func ToResponse(o order.Order) OrderResponse {
	resp := OrderResponse{
		OrderUID:        o.OrderUID,
		TrackNumber:     o.TrackNumber,
		Entry:           o.Entry,
		Locale:          o.Locale,
		CustomerID:      o.CustomerId,
		DeliveryService: o.DeliveryService,
		DateCreated:     o.DateCreated.Format("2006-01-02T15:04:05Z07:00"), // RFC3339

		Delivery: DeliveryResponse{
			Name:    o.Delivery.Name,
			Phone:   o.Delivery.Phone,
			Zip:     o.Delivery.Zip,
			City:    o.Delivery.City,
			Address: o.Delivery.Address,
			Region:  o.Delivery.Region,
			Email:   o.Delivery.Email,
		},
		Payment: PaymentResponse{
			Transaction:  o.Payment.Transaction,
			Currency:     o.Payment.Currency,
			Provider:     o.Payment.Provider,
			Amount:       o.Payment.Amount,
			PaymentDt:    o.Payment.PaymentDt.Format("2006-01-02T15:04:05Z07:00"),
			Bank:         o.Payment.Bank,
			DeliveryCost: o.Payment.DeliveryCost,
			GoodsTotal:   o.Payment.GoodsTotal,
			CustomFee:    o.Payment.CustomFee,
		},
		Items: make([]ItemResponse, 0, len(o.Items)),
	}

	for _, it := range o.Items {
		resp.Items = append(resp.Items, ItemResponse{
			ChrtID:      it.ChrtId,
			TrackNumber: it.TrackNumber,
			Price:       it.Price,
			Rid:         it.Rid,
			Name:        it.Name,
			Sale:        it.Sale,
			Size:        it.Size,
			TotalPrice:  it.TotalPrice,
			NmID:        it.NmId,
			Brand:       it.Brand,
			Status:      it.Status,
		})
	}
	return resp
}
