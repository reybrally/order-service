package repo

import (
	"context"
	"errors"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
	"time"
)

const qFindFullOrderByUID = `
SELECT
  o.order_uid,
  o.track_number,
  o.entry,
  o.locale,
  o.internal_signature,
  o.customer_id,
  o.delivery_service,
  o.shard_key,
  o.sm_id,
  o.date_created,
  o.oof_shard,

  d.delivery_name,
  d.phone,
  d.zip,
  d.city,
  d.address,
  d.region,
  d.email,

  p.transaction,
  p.request_id,
  p.currency,
  p.provider,
  p.amount,
  p.payment_dt,
  p.bank,
  p.delivery_cost,
  p.goods_total,
  p.custom_fee,

  i.chrt_id,
  i.track_number AS item_track_number,
  i.price,
  i.rid,
  i.item_name  AS name,
  i.sale,
  i.item_size  AS size,
  i.total_price,
  i.nm_id,
  i.brand,
  i.status
FROM orders o
LEFT JOIN deliveries   d ON d.order_uid = o.order_uid
LEFT JOIN payments     p ON p.order_uid = o.order_uid
LEFT JOIN order_items  i ON i.order_uid = o.order_uid
WHERE o.order_uid = $1;
`

func (r OrderRepo) GetOrder(ctx context.Context, uid string) (order.Order, error) {
	logging.LogInfo("Attempting to fetch order by order_uid", logrus.Fields{"order_uid": uid})
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.repo.Query(ctx, qFindFullOrderByUID, uid)
	if err != nil {
		logging.LogError("Error executing query to fetch order", err, logrus.Fields{"order_uid": uid})
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			logging.LogError("Context timeout or cancellation while fetching order", err, logrus.Fields{"order_uid": uid})
			return order.Order{}, orders.ErrTimeout
		}
		return order.Order{}, err
	}
	defer rows.Close()
	var (
		found bool
		out   order.Order
	)

	for rows.Next() {
		found = true

		var (
			orderUID, trackNumber, entry, locale, internalSig, customerID, deliveryService, shardKey string
			smID, oofShard                                                                           int64
			dateCreated                                                                              time.Time
		)

		var (
			dName, dPhone, dZip, dCity, dAddress, dRegion, dEmail *string
		)

		var (
			pTransaction, pRequestID, pCurrency, pProvider, pBank *string
			pAmount, pDeliveryCost, pGoodsTotal, pCustomFee       *int64
			pPaymentDt                                            *time.Time
		)

		var (
			iChrtID, iItemTrackNumber, iRid, iName, iNmID, iBrand *string
			iPrice, iSale, iSize, iTotalPrice, iStatus            *int64
		)

		if err := rows.Scan(
			&orderUID, &trackNumber, &entry, &locale, &internalSig, &customerID, &deliveryService, &shardKey, &smID, &dateCreated, &oofShard,
			&dName, &dPhone, &dZip, &dCity, &dAddress, &dRegion, &dEmail,
			&pTransaction, &pRequestID, &pCurrency, &pProvider, &pAmount, &pPaymentDt, &pBank, &pDeliveryCost, &pGoodsTotal, &pCustomFee,
			&iChrtID, &iItemTrackNumber, &iPrice, &iRid, &iName, &iSale, &iSize, &iTotalPrice, &iNmID, &iBrand, &iStatus,
		); err != nil {
			logging.LogError("Error scanning row for order", err, logrus.Fields{"order_uid": uid})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				logging.LogError("Context timeout or cancellation while scanning row", err, logrus.Fields{"order_uid": uid})
				return order.Order{}, orders.ErrTimeout
			}
			return order.Order{}, err
		}

		if out.OrderUID == "" {
			out = order.Order{
				OrderUID:          orderUID,
				TrackNumber:       trackNumber,
				Entry:             entry,
				Locale:            locale,
				InternalSignature: internalSig,
				CustomerId:        customerID,
				DeliveryService:   deliveryService,
				ShardKey:          shardKey,
				SmId:              smID,
				DateCreated:       dateCreated,
				OofShard:          oofShard,
				Delivery: order.Delivery{
					Name:    derefStr(dName),
					Phone:   derefStr(dPhone),
					Zip:     derefStr(dZip),
					City:    derefStr(dCity),
					Address: derefStr(dAddress),
					Region:  derefStr(dRegion),
					Email:   derefStr(dEmail),
				},
				Payment: order.Payment{
					Transaction:  derefStr(pTransaction),
					RequestId:    derefStr(pRequestID),
					Currency:     derefStr(pCurrency),
					Provider:     derefStr(pProvider),
					Amount:       derefI64(pAmount),
					PaymentDt:    derefTime(pPaymentDt),
					Bank:         derefStr(pBank),
					DeliveryCost: derefI64(pDeliveryCost),
					GoodsTotal:   derefI64(pGoodsTotal),
					CustomFee:    derefI64(pCustomFee),
				},
				Items: make([]order.Item, 0, 8),
			}
		}

		if iChrtID != nil {
			out.Items = append(out.Items, order.Item{
				ChrtId:      derefStr(iChrtID),
				TrackNumber: derefStr(iItemTrackNumber),
				Price:       derefI64(iPrice),
				Rid:         derefStr(iRid),
				Name:        derefStr(iName),
				Sale:        derefI64(iSale),
				Size:        derefI64(iSize),
				TotalPrice:  derefI64(iTotalPrice),
				NmId:        derefStr(iNmID),
				Brand:       derefStr(iBrand),
				Status:      derefI64(iStatus),
			})
		}
	}
	if err := rows.Err(); err != nil {
		logging.LogError("Error iterating over rows", err, logrus.Fields{"order_uid": uid})
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			logging.LogError("Context timeout or cancellation while iterating rows", err, logrus.Fields{"order_uid": uid})
			return order.Order{}, orders.ErrTimeout
		}
		return order.Order{}, err
	}
	if !found {
		logging.LogError("Order not found", nil, logrus.Fields{"order_uid": uid})
		return order.Order{}, orders.ErrNotFound
	}

	logging.LogInfo("Order fetched successfully", logrus.Fields{"order_uid": uid})
	return out, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefI64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}
