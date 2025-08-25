// cmd/seeder/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	ctx := context.Background()

	// Берём конфиг из .env (у тебя всё уже есть в docker-compose)
	host := getenv("DB_HOST", "127.0.0.1")
	port := getenv("DB_PORT", "55432")
	user := getenv("DB_USER", "postgres")
	pass := getenv("DB_PASSWORD", "postgres")
	db := getenv("DB_NAME", "orders_db")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, db)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	rand.Seed(time.Now().UnixNano())

	const ordersN = 1000
	minItems, maxItems := 1, 5

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("begin: %v", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// ---------- вставляем orders батчем ----------
	{
		batch := &pgx.Batch{}
		now := time.Now()
		for i := 1; i <= ordersN; i++ {
			orderUID := fmt.Sprintf("ORD-%06d", i)
			track := fmt.Sprintf("TRK-%06d", i)
			entry := "WEB"
			locale := "ru"
			sig := "seed"
			customer := fmt.Sprintf("cust-%d", 1000+i)
			deliveryService := []string{"cdek", "boxberry", "pochta"}[rand.Intn(3)]
			shard := fmt.Sprintf("%d", 1+rand.Intn(4))
			smID := 10 + rand.Intn(90)
			dateCreated := now.Add(-time.Duration(rand.Intn(86400)) * time.Second)
			oof := 1 + rand.Intn(5)

			batch.Queue(`
				INSERT INTO orders (
					order_uid, track_number, entry, locale, internal_signature,
					customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
				ON CONFLICT (order_uid) DO NOTHING
			`,
				orderUID, track, entry, locale, sig, customer,
				deliveryService, shard, smID, dateCreated, oof,
			)
		}

		br := tx.SendBatch(ctx, batch)
		if err = br.Close(); err != nil {
			log.Fatalf("orders batch close: %v", err)
		}
	}

	// ---------- вставляем items батчем ----------
	{
		batch := &pgx.Batch{}
		for i := 1; i <= ordersN; i++ {
			orderUID := fmt.Sprintf("ORD-%06d", i)
			itemsCount := minItems + rand.Intn(maxItems-minItems+1)
			for j := 1; j <= itemsCount; j++ {
				chrtID := fmt.Sprintf("%d", 10000+i*10+j) // string в домене
				track := fmt.Sprintf("TRK-%06d", i)
				price := int64(1000 + rand.Intn(300000))
				sale := int64(rand.Intn(30000))
				size := int64([]int{36, 38, 40, 42, 44, 46, 48, 50}[rand.Intn(8)])
				total := price - sale
				if total < 0 {
					total = 0
				}
				rid := fmt.Sprintf("RID-%06d-%02d", i, j)
				name := []string{"Кроссовки", "Футболка", "Наушники", "Кружка", "Рюкзак"}[rand.Intn(5)]
				nmID := fmt.Sprintf("%d", 5000+rand.Intn(5000)) // string в домене
				brand := []string{"Nike", "Adidas", "Puma", "Reebok", "Apple", "Xiaomi"}[rand.Intn(6)]
				status := int64(1)

				batch.Queue(`
					INSERT INTO order_items (
						order_uid, chrt_id, track_number, price, rid, name, sale,
						size, total_price, nm_id, brand, status
					) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
					ON CONFLICT (order_uid, chrt_id) DO NOTHING
				`,
					orderUID, chrtID, track, price, rid, name, sale,
					size, total, nmID, brand, status,
				)
			}
		}

		br := tx.SendBatch(ctx, batch)
		if err = br.Close(); err != nil {
			log.Fatalf("items batch close: %v", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}

	log.Printf("✅ seeded %d orders (with items)\n", ordersN)
}
