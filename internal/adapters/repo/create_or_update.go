package repo

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
)

const qOrders = `INSERT INTO orders (
		order_uid, track_number, entry, locale, internal_signature, customer_id,
		delivery_service, shard_key, sm_id, date_created, oof_shard
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	ON CONFLICT (order_uid) DO UPDATE SET
	track_number       = EXCLUDED.track_number,
		entry              = EXCLUDED.entry,
		locale             = EXCLUDED.locale,
	internal_signature = EXCLUDED.internal_signature,
		customer_id        = EXCLUDED.customer_id,
		delivery_service   = EXCLUDED.delivery_service,
		shard_key          = EXCLUDED.shard_key,
		sm_id              = EXCLUDED.sm_id,
		date_created       = EXCLUDED.date_created,
		oof_shard          = EXCLUDED.oof_shard
	RETURNING
	order_uid, track_number, entry, locale, internal_signature, customer_id,
		delivery_service, shard_key, sm_id, date_created, oof_shard;
	`

func (r *OrderRepo) CreateOrUpdateOrder(ctx context.Context, o order.Order) (order.Order, error) {
	tx, err := r.repo.Begin(ctx)
	if err != nil {
		return order.Order{}, err
	}
	defer tx.Rollback(ctx)

	var orderRow OrderRow
	err = tx.QueryRow(ctx, qOrders, o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerId, o.DeliveryService,
		o.ShardKey, o.SmId, o.DateCreated, o.OofShard).Scan(&orderRow.OrderUID, &orderRow.TrackNumber, &orderRow.Entry, &orderRow.Locale,
		&orderRow.InternalSignature, &orderRow.CustomerId, &orderRow.DeliveryService, &orderRow.ShardKey, &orderRow.SmId, &orderRow.DateCreated, &orderRow.OofShard)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return order.Order{}, orders.ErrUnexpected
		}
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			switch pgerr.Code {
			case "23514":
				return order.Order{}, orders.ErrInvalidData
			case "40001":
				return order.Order{}, orders.ErrRetry
			}
		}
		return order.Order{}, err
	}
	const qDelivery = `
	INSERT INTO order_deliveries (order_uid, name, phone, zip, city, address, region, email)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (order_uid) DO UPDATE SET
	                                                                       delivery_name = EXCLUDED.name,
	                                                                       phone = EXCLUDED.phone,
	                                                                       zip = EXCLUDED.zip,
	                                                                       city = EXCLUDED.city,
	                                                                       address = EXCLUDED.address,
	                                                                       region = EXCLUDED.region,
	                                                                       email = EXCLUDED.email
	RETURNING delivery_name, phone, zip, city, address, region, email;`
	var deliveryRow DeliveryRow
	err = tx.QueryRow(ctx, qDelivery, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email,
	).Scan(&deliveryRow.Name, &deliveryRow.Phone, &deliveryRow.Zip, &deliveryRow.City, &deliveryRow.Address, &deliveryRow.Region, &deliveryRow.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return order.Order{}, orders.ErrUnexpected
		}
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			switch pgerr.Code {
			case "23503": // FK violation
				return order.Order{}, orders.ErrInvalidReference
			case "23514":
				return order.Order{}, orders.ErrInvalidData
			}
		}
		return order.Order{}, err
	}
	const qItem = `
INSERT INTO order_items (
    order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (order_uid, chrt_id) DO UPDATE
SET
    track_number = EXCLUDED.track_number,
    price        = EXCLUDED.price,
    rid          = EXCLUDED.rid,
    item_name         = EXCLUDED.name,   
    sale         = EXCLUDED.sale,
    item_size         = EXCLUDED.size,   
    total_price  = EXCLUDED.total_price,
    nm_id        = EXCLUDED.nm_id,
    brand        = EXCLUDED.brand,
    status       = EXCLUDED.status
RETURNING chrt_id, track_number, price, rid, item_name, sale, item_size, total_price, nm_id, brand, status;

`
	ItemRows := make([]ItemRow, 0)
	for i := range o.Items {
		var itemRow ItemRow

		row := tx.QueryRow(
			ctx, qItem,
			o.OrderUID,
			o.Items[i].ChrtId,
			o.Items[i].TrackNumber,
			o.Items[i].Price,
			o.Items[i].Rid,
			o.Items[i].Name,
			o.Items[i].Sale,
			o.Items[i].Size,
			o.Items[i].TotalPrice,
			o.Items[i].NmId,
			o.Items[i].Brand,
			o.Items[i].Status,
		)

		if err := row.Scan(
			&itemRow.ChrtId,
			&itemRow.TrackNumber,
			&itemRow.Price,
			&itemRow.Rid,
			&itemRow.Name,
			&itemRow.Sale,
			&itemRow.Size,
			&itemRow.TotalPrice,
			&itemRow.NmId,
			&itemRow.Brand,
			&itemRow.Status,
		); err != nil {

			if errors.Is(err, pgx.ErrNoRows) {
				return order.Order{}, orders.ErrUnexpected
			}

			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return order.Order{}, orders.ErrTimeout
			}

			var pgerr *pgconn.PgError
			if errors.As(err, &pgerr) {
				switch pgerr.Code {
				case "23503":
					return order.Order{}, orders.ErrInvalidReference
				case "23514":
					return order.Order{}, orders.ErrInvalidData
				case "23505":
					return order.Order{}, orders.ErrConflict
				case "23502":
					return order.Order{}, orders.ErrInvalidData
				case "22001":
					return order.Order{}, orders.ErrInvalidData
				case "22P02":
					return order.Order{}, orders.ErrInvalidData
				case "40001":
					return order.Order{}, orders.ErrRetryable
				case "40P01":
					return order.Order{}, orders.ErrRetryable
				}
			}

			return order.Order{}, err
		}
		ItemRows = append(ItemRows, itemRow)
	}

	return orderRow.ToDomain(), nil
}
