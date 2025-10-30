package main

import (
	"context"
	"encoding/json"
	"github.com/reybrally/order-service/internal/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	segmentio "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/reybrally/order-service/internal/adapters/cache"
	httpHandlers "github.com/reybrally/order-service/internal/adapters/http/handlers"
	kaf "github.com/reybrally/order-service/internal/adapters/kafka"
	repoPkg "github.com/reybrally/order-service/internal/adapters/repo"
	"github.com/reybrally/order-service/internal/logging"
	svcPkg "github.com/reybrally/order-service/internal/services"
)

func main() {
	cfg := config.Load()
	logging.InitLogger()
	logging.LogInfo("starting order-service", logrus.Fields{
		"pid":  os.Getpid(),
		"port": getenv("PORT", "8080"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := mustPG(ctx, cfg)
	defer pool.Close()

	repo := repoPkg.NewOrderRepo(pool)
	var cacheService cache.Cache
	if cfg.App.CacheBackend == "redis" {
		cacheService = cache.NewRedisCache(cache.RedisConfig{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
			Prefix:   cfg.Redis.Prefix,
			TTL:      cfg.Redis.TTL,
		})
		logging.LogInfo("redis cache enabled", logrus.Fields{"addr": cfg.Redis.Addr, "ttl": cfg.Redis.TTL.String()})
	} else {
		cacheService = cache.NewCacheService(1000)
		logging.LogInfo("lru cache enabled", logrus.Fields{"capacity": 1000})
	}

	prod := mustKafkaProducer(cfg)
	defer prod.Close()

	eventsTopic := getenv("ORDERS_EVENTS_TOPIC", "orders-events")

	svc := svcPkg.NewOrderService(repo, cacheService, prod, eventsTopic)
	h := httpHandlers.NewOrderHandlers(svc)

	consumer := kaf.NewConsumer(kaf.ConsumerConfig{
		Brokers:           cfg.Kafka.Brokers,
		ClientID:          "order-service",
		MinBytes:          1 << 10,
		MaxBytes:          10 << 20,
		MaxWait:           100 * time.Millisecond,
		SessionTimeout:    10 * time.Second,
		RebalanceTimeout:  10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		StartOffset:       segmentio.FirstOffset,
		MaxRetries:        5,
		Backoff:           200 * time.Millisecond,
	})

	go func() {
		group := getenv("ORDERS_CONSUMER_GROUP", "order-service-cache-projector")
		logging.LogInfo("kafka consumer subscribing", logrus.Fields{
			"topic": eventsTopic, "group": group, "brokers": brokers(),
		})

		if err := consumer.Subscribe(ctx, eventsTopic, group, func(ctx context.Context, msg kaf.Message) error {
			switch msg.Envelope.EventType {
			case "order.upserted":
				var p kaf.OrderUpserted
				if err := json.Unmarshal(msg.Envelope.Payload, &p); err != nil {
					logging.LogError("cache-projector bad payload (upserted)", err, logrus.Fields{})
					return nil
				}
				ord, err := repo.GetOrder(ctx, p.OrderUID)
				if err != nil {
					return err
				}
				if err := cacheService.Set(p.OrderUID, ord); err != nil {
					return err
				}
				logging.LogInfo("cache-projector cached", logrus.Fields{"order_uid": p.OrderUID})
				return nil

			case "order.deleted":
				var p kaf.OrderDeleted
				if err := json.Unmarshal(msg.Envelope.Payload, &p); err != nil {
					logging.LogError("cache-projector bad payload (deleted)", err, logrus.Fields{})
					return nil
				}
				_ = cacheService.Delete(p.OrderUID)
				logging.LogInfo("cache-projector evicted", logrus.Fields{"order_uid": p.OrderUID})
				return nil

			default:
				return nil
			}
		}); err != nil {
			logging.LogError("kafka consumer stopped", err, logrus.Fields{
				"topic": eventsTopic, "group": group,
			})
		} else {
			logging.LogInfo("kafka consumer exited gracefully", logrus.Fields{
				"topic": eventsTopic, "group": group,
			})
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.StripSlashes, middleware.Timeout(5*time.Second))
	r.Get("/health", httpHandlers.HealthHandler)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			logging.LogError("readiness: db not ready", err, logrus.Fields{})
			http.Error(w, "db not ready: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	r.Route("/orders", func(r chi.Router) {
		r.Post("/", h.CreateOrUpdateOrder)
		r.Put("/", h.CreateOrUpdateOrder)
		r.Get("/search", h.SearchOrders)
		r.Get("/{id}", h.GetHandler)
		r.Delete("/{id}", h.DeleteHandler)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logging.LogInfo("http server listening", logrus.Fields{"addr": srv.Addr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.LogError("http server ListenAndServe failed", err, logrus.Fields{"addr": srv.Addr})
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop
	logging.LogInfo("shutdown signal received", logrus.Fields{"signal": sig.String()})

	if err := consumer.Close(); err != nil {
		logging.LogError("kafka consumer close failed", err, logrus.Fields{})
	} else {
		logging.LogInfo("kafka consumer closed", logrus.Fields{})
	}

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	if err := srv.Shutdown(shCtx); err != nil {
		logging.LogError("http server shutdown failed", err, logrus.Fields{})
	} else {
		logging.LogInfo("http server shutdown complete", logrus.Fields{})
	}
	logging.LogInfo("bye", logrus.Fields{})
}

func mustPG(ctx context.Context, cfg config.Config) *pgxpool.Pool {
	dbURL := os.Getenv("DATABASE_URL")
	fields := logrus.Fields{}
	if dbURL == "" {
		dbURL = "postgres://" + cfg.DB.User + ":" + cfg.DB.Password + "@" +
			cfg.DB.Host + ":" + cfg.DB.Port + "/" + cfg.DB.Name + "?sslmode=" + cfg.DB.SSLMode
		fields = logrus.Fields{
			"source":  "env/defaults",
			"host":    cfg.DB.Host,
			"port":    cfg.DB.Port,
			"db_name": cfg.DB.Name,
			"user":    cfg.DB.User,
			"sslmode": cfg.DB.SSLMode,
		}
	} else {
		fields = logrus.Fields{"source": "DATABASE_URL"}
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logging.LogError("pgxpool.New failed", err, fields)
		os.Exit(1)
	}
	logging.LogInfo("pgx pool created", fields)
	return pool
}

func mustKafkaProducer(cfg config.Config) kaf.Producer {
	p, err := kaf.NewProducer(kaf.ProducerConfig{
		Brokers:                cfg.Kafka.Brokers,
		ClientID:               "order-service",
		RequiredAcks:           segmentio.RequireAll,
		BatchBytes:             1 << 20,
		BatchTimeout:           50 * time.Millisecond,
		Compression:            segmentio.Snappy,
		Async:                  false,
		WriteTimeout:           5 * time.Second,
		AllowAutoTopicCreation: true,
	})
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	logging.LogInfo("kafka producer created", logrus.Fields{"brokers": cfg.Kafka.Brokers, "client_id": "order-service"})
	return p
}

func brokers() []string {
	return []string{getenv("KAFKA_BROKERS", "localhost:9092")}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
