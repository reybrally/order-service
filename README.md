# Order Service

**Order Service** - микросервис управления заказами, написанный на **Go**. 
Проект демонстрирует работу с **PostgreSQL**, **Kafka (Redpanda)**, **Redis-кэшированием**, **логированием (logrus)** и **REST API** на базе **Chi**.  
Используется **гексагональная архитектура** с чистым разделением домена, портов и адаптеров.

---

## Возможности

- Создание, обновление, удаление и поиск заказов через REST API
- **Kafka event-driven архитектура**
    - При создании заказа сервис публикует событие `order.upserted` в Kafka
    - Отдельный Kafka consumer ("cache projector") подписывается на события и обновляет кэш
- **Двухуровневое кэширование:**
    - LRU (in-memory) — для лёгкой локальной работы
    - Redis (через `CACHE_BACKEND=redis`) — для продакшн/докера
- **Подробное структурированное логирование** (`logrus`)
- Полностью контейнеризован через `docker-compose`
- Поддержка `.env` и централизованный конфиг-лоадер

---

## Архитектура

```
cmd/
 └── server/           — main entrypoint

internal/
 ├── adapters/         — внешние адаптеры
 │   ├── http/handlers — REST-эндпоинты (Chi)
 │   ├── kafka/        — Kafka producer/consumer
 │   ├── repo/         — PostgreSQL слой (pgx)
 │   └── cache/        — LRU и Redis-реализации
 │
 ├── app/orders/       — порты (интерфейсы)
 ├── domain/order/     — сущности домена
 ├── services/         — бизнес-логика
 ├── config/           — конфигурация (ENV loader)
 └── logging/          — логгер (logrus)
```

### Поток данных
1. Клиент отправляет запрос `POST /orders`
2. `OrderService` → сохраняет заказ в PostgreSQL
3. Публикует событие в Kafka (`orders-events`)
4. Kafka Consumer (cache-projector) ловит событие → тянет заказ → обновляет Redis/LRU кэш

---

## Технологии

| Компонент        | Используется                                   |
|------------------|------------------------------------------------|
| **Язык**         | Go 1.22+                                       |
| **Web**          | Chi Router + net/http                          |
| **БД**           | PostgreSQL + Goose migrations                  |
| **Кэш**          | Redis (go-redis/v9) / LRU                      |
| **Брокер событий** | Redpanda (Kafka-совместимая)                 |
| **Логирование**  | logrus (JSON-формат)                           |
| **Конфигурация** | через `internal/config` + ENV переменные       |
| **Контейнеризация** | Docker + docker-compose                      |

---

## Основные команды


Контейнеры:
- server - основной Go-сервис (`order-service`)
- postgres - БД
- redpanda - Kafka брокер
- redis - кэш
- migrator/seeder - миграции и тестовые данные
- topic-init - создаёт Kafka-топик `orders-events`

---

## Примеры API

### Создание заказа

```json
{
  "track_number": "WBILMTEST-POST-001",
  "entry": "WBIL",
  "locale": "en",
  "customer_id": "cust-001",
  "delivery_service": "DHL",
  "shard_key": "9",
  "sm_id": 777,
  "oof_shard": 1,
  "delivery": {
    "name": "John Doe",
    "phone": "+1-555-0100",
    "zip": "94105",
    "city": "San Francisco",
    "address": "135 5th St",
    "region": "CA",
    "email": "john.doe@example.com"
  },
  "payment": {
    "transaction": "tx-POST-001",
    "request_id": "req-POST-001",
    "currency": "USD",
    "provider": "visa",
    "amount": 350,
    "payment_dt": "2021-11-26T07:22:19Z",
    "bank": "BigBank",
    "delivery_cost": 50,
    "goods_total": 300,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": "ch-1",
      "track_number": "WBILMTEST-POST-001",
      "price": 200,
      "rid": "rid-1",
      "item_name": "item-1",
      "sale": 0,
      "item_size": 10,
      "total_price": 200,
      "nm_id": "nm-1",
      "brand": "brand-A",
      "status": 1
    }
  ]
}
```

---

## Логирование

```json
{
  "time": "2025-10-30T06:54:55+03:00",
  "level": "info",
  "msg": "Order successfully created or updated",
  "order_uid": "3cb63b3f-e181-47ab-92c2-3b3e54714920"
}
```

---
