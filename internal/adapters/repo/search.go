package repo

import (
	"context"
	"fmt"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
	"strings"
)

var sortWhitelist = map[string]string{
	"date_created": "o.date_created",
	"customer_id":  "o.customer_id",
	"track_number": "o.track_number",
	"amount":       "p.amount",
}

func (r *OrderRepo) SearchOrders(ctx context.Context, f orders.SearchFilters, p orders.PageRequest) ([]order.Order, error) {
	logging.LogInfo("Starting order search", logrus.Fields{
		"filters":      f,
		"page_request": p,
	})

	var (
		sb   strings.Builder
		args []any
		n    = 1
	)

	sb.WriteString(`
    SELECT o.order_uid
    FROM orders o
    LEFT JOIN payments p ON p.order_uid = o.order_uid
    WHERE 1=1
  `)

	if f.CreatedFrom != nil {
		sb.WriteString(fmt.Sprintf(" AND o.date_created >= $%d", n))
		args = append(args, *f.CreatedFrom)
		n++
	}
	if f.CreatedTo != nil {
		sb.WriteString(fmt.Sprintf(" AND o.date_created < $%d", n))
		args = append(args, *f.CreatedTo)
		n++
	}
	if f.OrderUID != nil {
		sb.WriteString(fmt.Sprintf(" AND o.order_uid = $%d", n))
		args = append(args, *f.OrderUID)
		n++
	}
	if f.TrackNumber != nil {
		sb.WriteString(fmt.Sprintf(" AND o.track_number ILIKE $%d", n))
		args = append(args, "%"+*f.TrackNumber+"%")
		n++
	}
	if f.CustomerID != nil {
		sb.WriteString(fmt.Sprintf(" AND o.customer_id = $%d", n))
		args = append(args, *f.CustomerID)
		n++
	}
	if f.Provider != nil {
		sb.WriteString(fmt.Sprintf(" AND p.provider = $%d", n))
		args = append(args, *f.Provider)
		n++
	}
	if f.Currency != nil {
		sb.WriteString(fmt.Sprintf(" AND p.currency = $%d", n))
		args = append(args, *f.Currency)
		n++
	}
	if f.Query != nil {
		sb.WriteString(fmt.Sprintf(`
      AND (
        o.order_uid    ILIKE $%[1]d OR
        o.track_number ILIKE $%[1]d OR
        o.customer_id  ILIKE $%[1]d OR
        p.transaction  ILIKE $%[1]d
      )`, n))
		args = append(args, "%"+*f.Query+"%")
		n++
	}

	col, ok := sortWhitelist[p.SortBy]
	if !ok {
		col = "o.date_created"
	}
	dir := strings.ToUpper(p.SortDir)
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	sb.WriteString(" ORDER BY " + col + " " + dir)

	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	sb.WriteString(fmt.Sprintf(" LIMIT $%d", n))
	args = append(args, p.Limit)
	n++
	if p.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET $%d", n))
		args = append(args, p.Offset)
		n++
	}

	rows, err := r.repo.Query(ctx, sb.String(), args...)
	if err != nil {
		logging.LogError("Error executing search query", err, logrus.Fields{"query": sb.String(), "args": args})
		return nil, err
	}
	defer rows.Close()

	var uids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			logging.LogError("Error scanning row in search query", err, logrus.Fields{"query": sb.String(), "args": args})
			return nil, err
		}
		uids = append(uids, id)
	}
	if err := rows.Err(); err != nil {
		logging.LogError("Error iterating over rows", err, logrus.Fields{"query": sb.String(), "args": args})
		return nil, err
	}

	logging.LogInfo("Found order_uids", logrus.Fields{"order_uids": uids})

	out := make([]order.Order, 0, len(uids))
	for _, id := range uids {
		o, err := r.GetOrder(ctx, id)
		if err != nil {
			logging.LogError("Error fetching order by order_uid", err, logrus.Fields{"order_uid": id})
			return nil, err
		}
		out = append(out, o)
	}
	logging.LogInfo("Search completed successfully", logrus.Fields{
		"found_orders": len(out),
	})
	return out, nil
}
