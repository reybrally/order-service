package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type App struct {
	Env          string
	CacheBackend string
}

type HTTP struct {
	Port string
}

type DB struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

type Kafka struct {
	Brokers []string
	Topic   string
	Group   string
	DLQ     string
}

type Redis struct {
	Addr     string
	Password string
	DB       int
	TTL      time.Duration
	Prefix   string
}

type Config struct {
	App   App
	HTTP  HTTP
	DB    DB
	Kafka Kafka
	Redis Redis
}

func Load() Config {
	return Config{
		App: App{
			Env:          getenv("APP_ENV", "dev"),
			CacheBackend: getenv("CACHE_BACKEND", "lru"),
		},
		HTTP: HTTP{
			Port: getenv("PORT", "8080"),
		},
		DB: DB{
			Host:     getenv("DB_HOST", "127.0.0.1"),
			Port:     getenv("DB_PORT", "55432"),
			Name:     getenv("DB_NAME", "orders_db"),
			User:     getenv("DB_USER", "postgres"),
			Password: getenv("DB_PASSWORD", "postgres"),
			SSLMode:  getenv("DB_SSLMODE", "disable"),
		},
		Kafka: Kafka{
			Brokers: splitCSV(getenv("KAFKA_BROKERS", "localhost:19092")),
			Topic:   getenv("ORDERS_EVENTS_TOPIC", "orders-events"),
			Group:   getenv("ORDERS_CONSUMER_GROUP", "order-service-cache-projector"),
			DLQ:     getenv("ORDERS_DLQ_TOPIC", "orders-events-dlq"),
		},
		Redis: Redis{
			Addr:     getenv("REDIS_ADDR", "localhost:6379"),
			Password: getenv("REDIS_PASSWORD", ""),
			DB:       atoi(getenv("REDIS_DB", "0")),
			TTL:      parseDuration(getenv("REDIS_TTL", "10m")),
			Prefix:   getenv("REDIS_PREFIX", "order:"),
		},
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}
