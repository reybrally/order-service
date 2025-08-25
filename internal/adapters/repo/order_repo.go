package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reybrally/order-service/internal/domain/order"
	"time"
)

type DeliveryRow struct {
	OrderUID string
	Name     string
	Phone    string
	Zip      string
	City     string
	Address  string
	Region   string
	Email    string
}
type ItemRow struct {
	OrderUID    string
	ChrtId      string
	TrackNumber string
	Price       int64
	Rid         string
	Name        string
	Sale        int64
	Size        int64
	TotalPrice  int64
	NmId        string
	Brand       string
	Status      int64
}

type OrderRepo struct {
	repo *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo { return &OrderRepo{repo: pool} }

type OrderRow struct {
	OrderUID          string
	TrackNumber       string
	Entry             string
	Locale            string
	InternalSignature string
	CustomerId        string
	DeliveryService   string
	ShardKey          string
	SmId              int64
	DateCreated       time.Time
	OofShard          int64
}

func (o *OrderRow) ToDomain() order.Order {
	return order.Order{OrderUID: o.OrderUID, TrackNumber: o.TrackNumber,
		Entry: o.Entry, Locale: o.Locale, InternalSignature: o.InternalSignature,
		CustomerId: o.CustomerId, DeliveryService: o.DeliveryService, ShardKey: o.ShardKey,
		SmId: o.SmId, DateCreated: o.DateCreated, OofShard: o.OofShard,
	}
}
