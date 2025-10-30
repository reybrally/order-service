package repo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reybrally/order-service/internal/adapters/repo"
	app "github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(),
		"user=postgres password=postgres dbname=orders_db host=127.0.0.1 port=55432 sslmode=disable")
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	return pool
}

func newTestRepo(t *testing.T) (*repo.OrderRepo, *pgxpool.Pool) {
	pool := newTestPool(t)
	t.Cleanup(func() { pool.Close() })
	r := repo.NewOrderRepo(pool)
	truncateAll(t, pool)
	return r, pool
}

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE order_items RESTART IDENTITY CASCADE;
		TRUNCATE TABLE payments     RESTART IDENTITY CASCADE;
		TRUNCATE TABLE deliveries   RESTART IDENTITY CASCADE;
		TRUNCATE TABLE orders       RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("truncateAll: %v", err)
	}
}

func makeOrder(uid string, withTwoItems bool) order.Order {
	items := []order.Item{
		{
			ChrtId:      "ch1",
			TrackNumber: "TRK-" + uid,
			Price:       100,
			Rid:         "rid-1",
			Name:        "item-1",
			Sale:        0,
			Size:        1,
			TotalPrice:  100,
			NmId:        "nm-1",
			Brand:       "brand-1",
			Status:      1,
		},
	}
	if withTwoItems {
		items = append(items, order.Item{
			ChrtId:      "ch2",
			TrackNumber: "TRK-" + uid,
			Price:       200,
			Rid:         "rid-2",
			Name:        "item-2",
			Sale:        0,
			Size:        2,
			TotalPrice:  200,
			NmId:        "nm-2",
			Brand:       "brand-2",
			Status:      1,
		})
	}
	return order.Order{
		OrderUID:          uid,
		TrackNumber:       "TRK-" + uid,
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "sig",
		CustomerId:        "cust-" + uid,
		DeliveryService:   "DHL",
		ShardKey:          "9",
		SmId:              123,
		DateCreated:       time.Now().UTC(),
		OofShard:          1,
		Delivery: order.Delivery{
			Name:    "John Doe",
			Phone:   "+123",
			Zip:     "12345",
			City:    "City",
			Address: "Street 1",
			Region:  "Region",
			Email:   fmt.Sprintf("john-%s@example.com", uid),
		},
		Payment: order.Payment{
			Transaction:  "tx-" + uid,
			RequestId:    "req-1",
			Currency:     "USD",
			Provider:     "visa",
			Amount:       300,
			PaymentDt:    time.Date(2021, 11, 26, 7, 22, 19, 0, time.UTC),
			Bank:         "Bank",
			DeliveryCost: 50,
			GoodsTotal:   250,
			CustomFee:    0,
		},
		Items: items,
	}
}

func mustSetDate(t *testing.T, pool *pgxpool.Pool, uid string, ts time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE orders SET date_created = $1 WHERE order_uid = $2`, ts, uid)
	if err != nil {
		t.Fatalf("mustSetDate(%s): %v", uid, err)
	}
}

func ptrTime(ti time.Time) *time.Time { return &ti }

func TestRepo_CreateOrUpdate_then_FindByUID_and_ItemsTrim(t *testing.T) {
	r, pool := newTestRepo(t)
	ctx := context.Background()

	o := makeOrder("uid-1", true)
	if _, err := r.CreateOrUpdateOrder(ctx, o); err != nil {
		t.Fatalf("CreateOrUpdateOrder(2 items): %v", err)
	}
	got, err := r.GetOrder(ctx, o.OrderUID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}

	o2 := makeOrder("uid-1", false)
	o2.Payment.Transaction = "tx-uid-1-v2"
	if _, err := r.CreateOrUpdateOrder(ctx, o2); err != nil {
		t.Fatalf("CreateOrUpdateOrder(1 item): %v", err)
	}
	got2, err := r.GetOrder(ctx, o.OrderUID)
	if err != nil {
		t.Fatalf("GetOrder(2): %v", err)
	}
	if len(got2.Items) != 1 {
		t.Fatalf("expected 1 item after trim, got %d", len(got2.Items))
	}
	if got2.Payment.Transaction != "tx-uid-1-v2" {
		t.Fatalf("payment not updated via upsert by order_uid, got %q", got2.Payment.Transaction)
	}

	if err := r.DeleteOrder(ctx, o.OrderUID); err != nil {
		t.Fatalf("DeleteOrder: %v", err)
	}
	if _, err = r.GetOrder(ctx, o.OrderUID); err == nil {
		t.Fatalf("expected not found after delete")
	}
	_ = pool
}

func TestRepo_CreateOrUpdate_WithoutItems_AllowsEmptyCart(t *testing.T) {
	r, _ := newTestRepo(t)
	ctx := context.Background()

	o := makeOrder("uid-empty", false)
	o.Items = nil
	if _, err := r.CreateOrUpdateOrder(ctx, o); err != nil {
		t.Fatalf("CreateOrUpdateOrder(empty items): %v", err)
	}
	got, err := r.GetOrder(ctx, o.OrderUID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(got.Items))
	}
}

func TestRepo_GetOrder_NotFound(t *testing.T) {
	r, _ := newTestRepo(t)
	ctx := context.Background()
	_, err := r.GetOrder(ctx, "no-such-uid")
	if err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestRepo_SearchOrders_SortingByDateCreated(t *testing.T) {
	r, pool := newTestRepo(t)
	ctx := context.Background()

	o1 := makeOrder("uid-2", false)
	o2 := makeOrder("uid-3", false)
	if _, err := r.CreateOrUpdateOrder(ctx, o1); err != nil {
		t.Fatalf("CreateOrUpdateOrder o1: %v", err)
	}
	if _, err := r.CreateOrUpdateOrder(ctx, o2); err != nil {
		t.Fatalf("CreateOrUpdateOrder o2: %v", err)
	}
	mustSetDate(t, pool, o1.OrderUID, time.Date(2021, 11, 26, 6, 22, 19, 0, time.UTC))
	mustSetDate(t, pool, o2.OrderUID, time.Date(2021, 11, 26, 7, 22, 19, 0, time.UTC))

	f := app.SearchFilters{
		CreatedFrom: ptrTime(time.Date(2021, 11, 26, 0, 0, 0, 0, time.UTC)),
		CreatedTo:   ptrTime(time.Date(2021, 11, 27, 0, 0, 0, 0, time.UTC)),
	}
	page := app.PageRequest{Limit: 10, Offset: 0, SortBy: "date_created", SortDir: "DESC"}

	got, err := r.SearchOrders(ctx, f, page)
	if err != nil {
		t.Fatalf("SearchOrders: %v", err)
	}
	if len(got) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(got))
	}
	if got[0].OrderUID != o2.OrderUID || got[1].OrderUID != o1.OrderUID {
		t.Fatalf("expected order by date desc: [%s,%s], got [%s,%s]",
			o2.OrderUID, o1.OrderUID, got[0].OrderUID, got[1].OrderUID)
	}
}

func TestRepo_SearchOrders_ByIndividualFilters(t *testing.T) {
	r, _ := newTestRepo(t)
	ctx := context.Background()

	a := makeOrder("A", false)
	b := makeOrder("B", false)
	c := makeOrder("C", false)
	a.Payment.Provider, a.Payment.Currency = "visa", "USD"
	b.Payment.Provider, b.Payment.Currency = "mc", "EUR"
	c.Payment.Provider, c.Payment.Currency = "paypal", "USD"

	if _, err := r.CreateOrUpdateOrder(ctx, a); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if _, err := r.CreateOrUpdateOrder(ctx, b); err != nil {
		t.Fatalf("create B: %v", err)
	}
	if _, err := r.CreateOrUpdateOrder(ctx, c); err != nil {
		t.Fatalf("create C: %v", err)
	}

	res, err := r.SearchOrders(ctx, app.SearchFilters{OrderUID: &a.OrderUID}, app.PageRequest{Limit: 10})
	if err != nil {
		t.Fatalf("Search by UID: %v", err)
	}
	if len(res) != 1 || res[0].OrderUID != a.OrderUID {
		t.Fatalf("expected 1 result with UID A, got %#v", res)
	}

	sub := "TRK-"
	res, err = r.SearchOrders(ctx, app.SearchFilters{TrackNumber: &sub}, app.PageRequest{Limit: 10})
	if err != nil || len(res) != 3 {
		t.Fatalf("Search by track substring: err=%v len=%d", err, len(res))
	}

	res, err = r.SearchOrders(ctx, app.SearchFilters{CustomerID: &b.CustomerId}, app.PageRequest{Limit: 10})
	if err != nil || len(res) != 1 || res[0].OrderUID != b.OrderUID {
		t.Fatalf("Search by customer: %v / %#v", err, res)
	}

	prov := "paypal"
	res, err = r.SearchOrders(ctx, app.SearchFilters{Provider: &prov}, app.PageRequest{Limit: 10})
	if err != nil || len(res) != 1 || res[0].OrderUID != c.OrderUID {
		t.Fatalf("Search by provider: %v / %#v", err, res)
	}

	cur := "USD"
	res, err = r.SearchOrders(ctx, app.SearchFilters{Currency: &cur}, app.PageRequest{Limit: 10})
	if err != nil || len(res) != 2 {
		t.Fatalf("Search by currency USD expected 2, got %d", len(res))
	}

	q := a.Payment.Transaction
	res, err = r.SearchOrders(ctx, app.SearchFilters{Query: &q}, app.PageRequest{Limit: 10})
	if err != nil || len(res) != 1 || res[0].OrderUID != a.OrderUID {
		t.Fatalf("Search by query(transaction): %v / %#v", err, res)
	}
}

func TestRepo_DeleteOrder_NotFound(t *testing.T) {
	r, _ := newTestRepo(t)
	ctx := context.Background()
	err := r.DeleteOrder(ctx, "no-such-uid")
	if err == nil {
		t.Fatalf("expected not found on delete")
	}
}
