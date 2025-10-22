package repo

import (
	"context"
	"errors"
	"github.com/reybrally/order-service/internal/app/orders"
)

const qDeleteOrder = `DELETE FROM orders WHERE order_uid = $1;`

func (r *OrderRepo) DeleteOrder(ctx context.Context, uid string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ct, err := r.repo.Exec(ctx, qDeleteOrder, uid)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return orders.ErrTimeout
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		return orders.ErrNotFound
	}
	return nil

}
