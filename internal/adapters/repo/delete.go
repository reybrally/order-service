package repo

import (
	"context"
	"errors"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
)

const qDeleteOrder = `DELETE FROM orders WHERE order_uid = $1;`

func (r *OrderRepo) DeleteOrder(ctx context.Context, uid string) error {
	logging.LogInfo("Attempting to delete order", logrus.Fields{"order_uid": uid})
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-ctx.Done():
		logging.LogError("Context was canceled or deadline exceeded", nil, logrus.Fields{"order_uid": uid})
		return orders.ErrTimeout
	default:
	}
	ct, err := r.repo.Exec(ctx, qDeleteOrder, uid)
	if err != nil {
		logging.LogError("Error executing DELETE query", err, logrus.Fields{"order_uid": uid})
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			logging.LogError("Context canceled or deadline exceeded during DELETE", err, logrus.Fields{"order_uid": uid})
			return orders.ErrTimeout
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		logging.LogError("Order not found to delete", nil, logrus.Fields{"order_uid": uid})
		return orders.ErrNotFound
	}

	logging.LogInfo("Order deleted successfully", logrus.Fields{"order_uid": uid})
	return nil

}
