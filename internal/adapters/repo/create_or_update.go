package repo

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
)

const (
	qOrders = `INSERT INTO orders (
    order_uid, track_number, entry, locale, internal_signature, customer_id,
    delivery_service, shard_key, sm_id, oof_shard
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (order_uid) DO UPDATE SET
    track_number       = EXCLUDED.track_number,
    entry              = EXCLUDED.entry,
    locale             = EXCLUDED.locale,
    internal_signature = EXCLUDED.internal_signature,
    customer_id        = EXCLUDED.customer_id,
    delivery_service   = EXCLUDED.delivery_service,
    shard_key          = EXCLUDED.shard_key,
    sm_id              = EXCLUDED.sm_id,
    oof_shard          = EXCLUDED.oof_shard
RETURNING
    order_uid, track_number, entry, locale, internal_signature, customer_id,
    delivery_service, shard_key, sm_id, date_created, oof_shard;`

	qDelivery = `
	INSERT INTO deliveries (
		order_uid, delivery_name, phone, zip, city, address, region, email
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	ON CONFLICT (order_uid) DO UPDATE SET
		delivery_name = EXCLUDED.delivery_name,
		phone         = EXCLUDED.phone,
		zip           = EXCLUDED.zip,
		city          = EXCLUDED.city,
		address       = EXCLUDED.address,
		region        = EXCLUDED.region,
		email         = EXCLUDED.email
	RETURNING delivery_name, phone, zip, city, address, region, email;`

	qItem = `
	INSERT INTO order_items (
		order_uid, chrt_id, track_number, price, rid, item_name, sale, item_size, total_price, nm_id, brand, status
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	ON CONFLICT (order_uid, chrt_id) DO UPDATE SET
		track_number = EXCLUDED.track_number,
		price        = EXCLUDED.price,
		rid          = EXCLUDED.rid,
		item_name    = EXCLUDED.item_name,
		sale         = EXCLUDED.sale,
		item_size    = EXCLUDED.item_size,
		total_price  = EXCLUDED.total_price,
		nm_id        = EXCLUDED.nm_id,
		brand        = EXCLUDED.brand,
		status       = EXCLUDED.status
	RETURNING chrt_id, track_number, price, rid, item_name, sale, item_size, total_price, nm_id, brand, status;`

	qPayment = `
INSERT INTO payments (
    order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (order_uid) DO UPDATE SET
    transaction   = EXCLUDED.transaction,
    request_id    = EXCLUDED.request_id,
    currency      = EXCLUDED.currency,
    provider      = EXCLUDED.provider,
    amount        = EXCLUDED.amount,
    payment_dt    = EXCLUDED.payment_dt,
    bank          = EXCLUDED.bank,
    delivery_cost = EXCLUDED.delivery_cost,
    goods_total   = EXCLUDED.goods_total,
    custom_fee    = EXCLUDED.custom_fee
RETURNING transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee;`
)

func (r *OrderRepo) CreateOrUpdateOrder(ctx context.Context, o order.Order) (order.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tx, err := r.repo.Begin(ctx)
	if err != nil {
		return order.Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orderRow OrderRow
	if err := tx.QueryRow(ctx, qOrders,
		o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerId,
		o.DeliveryService, o.ShardKey, o.SmId, o.OofShard,
	).Scan(&orderRow.OrderUID, &orderRow.TrackNumber, &orderRow.Entry, &orderRow.Locale,
		&orderRow.InternalSignature, &orderRow.CustomerId, &orderRow.DeliveryService,
		&orderRow.ShardKey, &orderRow.SmId, &orderRow.DateCreated, &orderRow.OofShard); err != nil {
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

	var deliveryRow DeliveryRow
	if err := tx.QueryRow(ctx, qDelivery,
		o.OrderUID,
		o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email,
	).Scan(&deliveryRow.Name, &deliveryRow.Phone, &deliveryRow.Zip, &deliveryRow.City,
		&deliveryRow.Address, &deliveryRow.Region, &deliveryRow.Email); err != nil {
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

	var pRow PaymentRow
	if err := tx.QueryRow(ctx, qPayment,
		o.OrderUID, o.Payment.Transaction, o.Payment.RequestId, o.Payment.Currency,
		o.Payment.Provider, o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank,
		o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee,
	).Scan(&pRow.Transaction, &pRow.RequestID, &pRow.Currency, &pRow.Provider, &pRow.Amount,
		&pRow.PaymentDt, &pRow.Bank, &pRow.DeliveryCost, &pRow.GoodsTotal, &pRow.CustomFee); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return order.Order{}, orders.ErrTimeout
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return order.Order{}, orders.ErrUnexpected
		}
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			switch pgerr.Code {
			case "23503":
				return order.Order{}, orders.ErrInvalidReference
			case "23514":
				return order.Order{}, orders.ErrInvalidData
			case "23502":
				return order.Order{}, orders.ErrInvalidData
			case "22001":
				return order.Order{}, orders.ErrInvalidData
			case "22P02":
				return order.Order{}, orders.ErrInvalidData
			case "40001",
				"40P01":
				return order.Order{}, orders.ErrRetryable
			case "23505":
				return order.Order{}, orders.ErrConflict
			}
		}
		return order.Order{}, err

	}
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
	if len(o.Items) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM order_items WHERE order_uid = $1`, o.OrderUID); err != nil {
			return order.Order{}, err
		}
	} else {
		ids := make([]string, 0, len(o.Items))
		for i := range o.Items {
			ids = append(ids, o.Items[i].ChrtId)
		}
		if _, err := tx.Exec(ctx, `
        DELETE FROM order_items
        WHERE order_uid = $1
          AND chrt_id NOT IN (SELECT unnest($2::text[]))
    `, o.OrderUID, ids); err != nil {
			return order.Order{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return order.Order{}, err
	}
	return orderRow.ToDomain(), nil
}
