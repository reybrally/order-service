package repo

import (
	"context"
	"github.com/reybrally/order-service/internal/domain/order"
)

const qFindOrderByUID = `
SELECT
    o.order_uid,
    o.track_number,
    o.entry,
    o.locale,
    o.internal_signature,
    o.customer_id,
    o.delivery_service,
    o.shardkey,
    o.sm_id,
    o.date_created,
    o.oof_shard,

    i.chrt_id,
    i.track_number AS item_track_number,
    i.price,
    i.rid,
    i.name,
    i.sale,
    i.size,
    i.total_price,
    i.nm_id,
    i.brand,
    i.status

FROM orders o
LEFT JOIN order_items i 
       ON o.order_uid = i.order_uid
WHERE o.order_uid = $1;
`


func (r OrderRepo) FindByUID(ctx context.Context, uid string) (order.Order, error) {
	rows, err :=
}
