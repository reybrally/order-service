package repo_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	r "github.com/reybrally/order-service/internal/adapters/repo"
	svc "github.com/reybrally/order-service/internal/app/orders"
	domain "github.com/reybrally/order-service/internal/domain/order"
)

/* ---------- setup helpers ---------- */

func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	// Если задан TEST_PG_DSN — используем его (локальный Postgres)
	if dsn := os.Getenv("TEST_PG_DSN"); dsn != "" {
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			t.Fatalf("pgxpool.New: %v", err)
		}
		t.Cleanup(func() { pool.Close() })
		applyMigrations(t, pool)
		return pool
	}

	// Иначе — поднимем Postgres через testcontainers
	pgC, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("orders"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pgC.Terminate(ctx) })

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable&pool_max_conns=5")
	if err != nil {
		t.Fatalf("conn string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	applyMigrations(t, pool)
	return pool
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	// Читаем SQL из testdata/001_init.sql (положи туда копию своей миграции)
	path := filepath.Join("testdata", "001_init.sql")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(b)

	// У Goose в файле есть Up/Down секции — возьмём только то, что между `-- +goose Up` и `-- +goose Down`
	up := extractGooseUp(sql)
	if strings.TrimSpace(up) == "" {
		up = sql
	}

	// Выполним как один батч
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
}

func extractGooseUp(all string) string {
	const upTag = "-- +goose Up"
	const downTag = "-- +goose Down"
	upIdx := strings.Index(all, upTag)
	if upIdx == -1 {
		return ""
	}
	rest := all[upIdx+len(upTag):]
	downIdx := strings.Index(rest, downTag)
	if downIdx == -1 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:downIdx])
}

/* ---------- fixtures ---------- */

func fixtureOrder(t *testing.T) domain.Order {
	t.Helper()
	return domain.Order{
		OrderUID:          "b563feb7b2b84b6test",
		TrackNumber:       "WBILMTESTTRACK",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerId:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmId:              99,
		DateCreated:       time.Date(2021, 11, 26, 6, 22, 19, 0, time.UTC),
		OofShard:          1,
		Delivery: domain.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: domain.Payment{
			Transaction:  "b563feb7b2b84b6test",
			RequestId:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    time.Unix(1637907727, 0).UTC(),
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []domain.Item{
			{
				ChrtId:      "9934930",
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        0,
				TotalPrice:  317,
				NmId:        "2389212",
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
			{
				ChrtId:      "1112223",
				TrackNumber: "WBILMTESTTRACK",
				Price:       100,
				Rid:         "rid-2",
				Name:        "Brush",
				Sale:        0,
				Size:        1,
				TotalPrice:  100,
				NmId:        "999",
				Brand:       "ACME",
				Status:      100,
			},
		},
	}
}

func TestRepo_CreateOrUpdate_then_FindByUID(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	repo := r.NewOrderRepo(pool)

	want := fixtureOrder(t)

	// 1) Create
	_, err := repo.CreateOrUpdateOrder(ctx, want)
	if err != nil {
		t.Fatalf("CreateOrUpdateOrder: %v", err)
	}

	// 2) Find
	got, err := repo.GetOrder(ctx, want.OrderUID)
	if err != nil {
		t.Fatalf("FindByUID: %v", err)
	}

	if got.OrderUID != want.OrderUID {
		t.Fatalf("OrderUID mismatch: got=%s want=%s", got.OrderUID, want.OrderUID)
	}
	if len(got.Items) != len(want.Items) {
		t.Fatalf("items len mismatch: got=%d want=%d", len(got.Items), len(want.Items))
	}
	if got.Payment.Currency != "USD" || got.Payment.Amount != 1817 {
		t.Fatalf("payment mismatch: got=%+v", got.Payment)
	}

	if _, err := repo.CreateOrUpdateOrder(ctx, want); err != nil {
		t.Fatalf("idempotent upsert failed: %v", err)
	}
}

func TestRepo_FindByUID_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	repo := r.NewOrderRepo(pool)

	_, err := repo.GetOrder(ctx, "no-such-id")
	if err == nil {
		t.Fatalf("expected error for not found")
	}
}

func TestRepo_DeleteOrder(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	repo := r.NewOrderRepo(pool)

	o := fixtureOrder(t)
	if _, err := repo.CreateOrUpdateOrder(ctx, o); err != nil {
		t.Fatalf("CreateOrUpdateOrder: %v", err)
	}

	if err := repo.DeleteOrder(ctx, o.OrderUID); err != nil {
		t.Fatalf("DeleteOrder: %v", err)
	}

	if err := repo.DeleteOrder(ctx, o.OrderUID); err == nil {
		t.Fatalf("expected not found on second delete")
	}
}

func TestRepo_SearchOrders(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	repo := r.NewOrderRepo(pool)

	// Подготовим данные: два разных заказа
	o1 := fixtureOrder(t)
	o2 := fixtureOrder(t)
	o2.OrderUID = "b563feb7b2b84b6test-2"
	o2.CustomerId = "another"
	o2.Payment.Transaction = "trx-2"
	o2.TrackNumber = "TRACK2"
	o2.DateCreated = o1.DateCreated.Add(1 * time.Hour)

	if _, err := repo.CreateOrUpdateOrder(ctx, o1); err != nil {
		t.Fatalf("insert o1: %v", err)
	}
	if _, err := repo.CreateOrUpdateOrder(ctx, o2); err != nil {
		t.Fatalf("insert o2: %v", err)
	}

	// 1) По customer_id
	results, err := repo.SearchOrders(ctx,
		svc.SearchFilters{CustomerID: &o1.CustomerId},
		svc.PageRequest{Limit: 10, SortBy: "date_created", SortDir: "desc"},
	)
	if err != nil {
		t.Fatalf("SearchOrders: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}

	// 2) По query (ищем по трек-номеру)
	q := "TRACK2"
	results, err = repo.SearchOrders(ctx,
		svc.SearchFilters{Query: &q},
		svc.PageRequest{Limit: 10},
	)
	if err != nil {
		t.Fatalf("SearchOrders by query: %v", err)
	}
	found := false
	for _, o := range results {
		if o.OrderUID == o2.OrderUID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find order %s by query", o2.OrderUID)
	}
}
