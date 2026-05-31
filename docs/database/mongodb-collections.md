# MongoDB Collections — Belanjayuk

Dokumentasi struktur dokumen untuk analytics-service.

MongoDB digunakan untuk menyimpan data analytics & event log
karena sifatnya yang flexible schema dan write-heavy.

---

## Database: `belanjayuk_analytics`

---

## Collection: `events`

Menyimpan semua business events yang dikonsumsi dari RabbitMQ.

```json
{
  "_id": "ObjectId",
  "event_id": "uuid-v4",
  "event_type": "order.created",
  "user_id": "uuid",
  "session_id": "string | null",
  "payload": {},
  "metadata": {
    "service": "order-service",
    "version": "1.0",
    "ip": "182.253.x.x",
    "user_agent": "Mozilla/5.0..."
  },
  "created_at": "ISODate"
}
```

**Event types yang di-consume:**

| event_type | Publisher | Payload |
|---|---|---|
| `user.registered` | auth-service | `{ user_id, email, method }` |
| `user.login` | auth-service | `{ user_id, ip, user_agent }` |
| `product.viewed` | product-service | `{ product_id, user_id, source }` |
| `product.searched` | product-service | `{ query, results_count, user_id }` |
| `cart.item_added` | cart-service | `{ user_id, variant_id, quantity }` |
| `cart.item_removed` | cart-service | `{ user_id, variant_id }` |
| `cart.abandoned` | cart-service | `{ user_id, items, total_value }` |
| `order.created` | order-service | `{ order_id, user_id, total, items_count }` |
| `order.cancelled` | order-service | `{ order_id, user_id, reason }` |
| `order.completed` | order-service | `{ order_id, user_id, total }` |
| `payment.success` | payment-service | `{ payment_id, order_id, amount, method }` |
| `payment.failed` | payment-service | `{ payment_id, order_id, reason }` |

**Indexes:**
```js
db.events.createIndex({ event_type: 1, created_at: -1 })
db.events.createIndex({ user_id: 1, created_at: -1 })
db.events.createIndex({ created_at: -1 }, { expireAfterSeconds: 7776000 }) // TTL 90 hari
db.events.createIndex({ event_id: 1 }, { unique: true })
```

---

## Collection: `daily_metrics`

Aggregasi harian untuk dashboard admin. Di-compute setiap malam via scheduled job.

```json
{
  "_id": "ObjectId",
  "date": "2025-01-15",
  "metrics": {
    "revenue": {
      "total": 15750000.00,
      "transaction_count": 127,
      "average_order_value": 124015.75
    },
    "orders": {
      "created": 145,
      "completed": 127,
      "cancelled": 12,
      "pending": 6
    },
    "users": {
      "new_registered": 89,
      "active": 432,
      "returning": 343
    },
    "products": {
      "total_views": 2841,
      "unique_viewers": 1205,
      "most_viewed": [
        { "product_id": "uuid", "name": "...", "views": 142 }
      ]
    },
    "conversion": {
      "cart_to_order_rate": 0.68,
      "order_to_paid_rate": 0.87,
      "overall_rate": 0.59
    },
    "payment_methods": {
      "bank_transfer": 45,
      "ewallet": 62,
      "credit_card": 15,
      "cod": 5
    }
  },
  "created_at": "ISODate",
  "updated_at": "ISODate"
}
```

**Indexes:**
```js
db.daily_metrics.createIndex({ date: -1 }, { unique: true })
```

---

## Collection: `product_analytics`

Performa per produk untuk keperluan rekomendasi dan merchandising.

```json
{
  "_id": "ObjectId",
  "product_id": "uuid",
  "period": "2025-01",
  "views": 1842,
  "unique_views": 934,
  "cart_adds": 287,
  "purchases": 198,
  "revenue": 29502000.00,
  "conversion_rate": 0.212,
  "average_rating": 4.7,
  "review_count": 156,
  "search_appearances": 3421,
  "search_clicks": 654,
  "updated_at": "ISODate"
}
```

**Indexes:**
```js
db.product_analytics.createIndex({ product_id: 1, period: -1 })
db.product_analytics.createIndex({ period: -1, revenue: -1 })
db.product_analytics.createIndex({ period: -1, conversion_rate: -1 })
```

---

## Collection: `user_behaviors`

Rekam jejak perilaku user untuk personalisasi & rekomendasi.

```json
{
  "_id": "ObjectId",
  "user_id": "uuid",
  "viewed_products": [
    {
      "product_id": "uuid",
      "viewed_at": "ISODate",
      "duration_seconds": 45
    }
  ],
  "searched_keywords": [
    {
      "keyword": "sepatu running",
      "searched_at": "ISODate"
    }
  ],
  "purchased_categories": ["uuid-cat-1", "uuid-cat-2"],
  "last_active_at": "ISODate",
  "updated_at": "ISODate"
}
```

**Indexes:**
```js
db.user_behaviors.createIndex({ user_id: 1 }, { unique: true })
db.user_behaviors.createIndex({ last_active_at: -1 })
```

---

## Collection: `search_logs`

Log pencarian untuk analisis keyword dan improve search relevance.

```json
{
  "_id": "ObjectId",
  "query": "sepatu running nike",
  "normalized_query": "sepatu running nike",
  "user_id": "uuid | null",
  "session_id": "string",
  "results_count": 24,
  "clicked_product_id": "uuid | null",
  "filters_applied": {
    "category_id": "uuid",
    "min_price": 100000,
    "max_price": 500000
  },
  "created_at": "ISODate"
}
```

**Indexes:**
```js
db.search_logs.createIndex({ query: 1, created_at: -1 })
db.search_logs.createIndex({ created_at: -1 }, { expireAfterSeconds: 2592000 }) // TTL 30 hari
```

---

## Collection: `notifications_log`

Log semua notifikasi yang dikirim oleh notification-service.

```json
{
  "_id": "ObjectId",
  "user_id": "uuid",
  "channel": "email | sms | push",
  "template": "order_confirmed",
  "recipient": "user@example.com | +628xx | fcm_token",
  "subject": "Pesanan #BJ-20250101-00001 dikonfirmasi",
  "status": "sent | failed | bounced",
  "error": "null | error message",
  "event_ref": "order.created",
  "sent_at": "ISODate",
  "created_at": "ISODate"
}
```

**Indexes:**
```js
db.notifications_log.createIndex({ user_id: 1, created_at: -1 })
db.notifications_log.createIndex({ status: 1, created_at: -1 })
db.notifications_log.createIndex({ created_at: -1 }, { expireAfterSeconds: 7776000 }) // TTL 90 hari
```

---

## Naming Convention

- Collection names: **snake_case**, plural
- Field names: **snake_case**
- ID references ke PostgreSQL: simpan sebagai `string` (UUID), bukan ObjectId
- Semua collection yang bersifat log wajib punya **TTL index** untuk auto-cleanup
- Field `created_at` wajib ada di semua collection

---

## Contoh Query Umum (Go + mongo-driver)

```go
// Get daily metrics untuk 7 hari terakhir
filter := bson.D{
    {"date", bson.D{{"$gte", sevenDaysAgo}}},
}
opts := options.Find().SetSort(bson.D{{"date", -1}})
cursor, err := db.Collection("daily_metrics").Find(ctx, filter, opts)

// Get top produk bulan ini
pipeline := mongo.Pipeline{
    {{"$match", bson.D{{"period", currentPeriod}}}},
    {{"$sort", bson.D{{"revenue", -1}}}},
    {{"$limit", 10}},
}
cursor, err := db.Collection("product_analytics").Aggregate(ctx, pipeline)
```