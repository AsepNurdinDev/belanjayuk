# Redis Key Patterns — BelanjayukK

Dokumentasi semua key pattern yang digunakan di Redis.

Format: `{prefix}:{identifier}` → value type (TTL)

---

## Auth Service

### Session & Token

| Key Pattern | Type | TTL | Value | Keterangan |
|---|---|---|---|---|
| `session:{user_id}` | Hash | 7 hari | `{token, user_agent, ip, created_at}` | Data session aktif |
| `refresh_token:{token}` | String | 7 hari | `{user_id}` | Mapping refresh token ke user |
| `blacklist:token:{token}` | String | Sisa expire token | `"1"` | Token yang sudah logout |
| `otp:{user_id}:{type}` | String | 5 menit | `{6-digit code}` | OTP untuk verifikasi email / reset password |
| `login_attempt:{ip}` | Counter | 15 menit | `{count}` | Rate limit login per IP |

**Contoh:**
```
session:550e8400-e29b-41d4-a716-446655440000
→ Hash {
    token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    user_agent: "Mozilla/5.0...",
    ip: "182.253.x.x",
    created_at: "2025-01-01T00:00:00Z"
  }
  TTL: 604800 (7 hari)

otp:550e8400-e29b-41d4-a716-446655440000:email_verification
→ "847291"
  TTL: 300 (5 menit)
```

---

## Cart Service

### Cart Data

| Key Pattern | Type | TTL | Value | Keterangan |
|---|---|---|---|---|
| `cart:user:{user_id}` | Hash | 30 hari | `{variant_id: quantity}` | Cart user login |
| `cart:guest:{session_id}` | Hash | 7 hari | `{variant_id: quantity}` | Cart guest (belum login) |
| `cart:lock:{user_id}` | String | 5 detik | `"1"` | Distributed lock saat update cart |

**Contoh:**
```
cart:user:550e8400-e29b-41d4-a716-446655440000
→ Hash {
    "variant-uuid-1": "2",
    "variant-uuid-2": "1",
    "variant-uuid-3": "5"
  }
  TTL: 2592000 (30 hari)

cart:guest:sess_abc123xyz
→ Hash {
    "variant-uuid-1": "1"
  }
  TTL: 604800 (7 hari)
```

**Catatan:** Saat guest login, `cart:guest:{session_id}` di-merge ke `cart:user:{user_id}` kemudian key guest dihapus.

---

## Product Service

### Cache Produk

| Key Pattern | Type | TTL | Value | Keterangan |
|---|---|---|---|---|
| `product:{product_id}` | String (JSON) | 1 jam | Product detail object | Cache detail produk |
| `product:slug:{slug}` | String | 1 jam | `{product_id}` | Lookup product_id dari slug |
| `products:list:{hash}` | String (JSON) | 15 menit | Array products | Cache hasil list/filter |
| `category:tree` | String (JSON) | 6 jam | Category tree | Cache tree kategori |

**Contoh:**
```
product:550e8400-e29b-41d4-a716-446655440000
→ JSON { id, name, price, variants, images, ... }
  TTL: 3600 (1 jam)

product:slug:sepatu-running-nike-air-max
→ "550e8400-e29b-41d4-a716-446655440000"
  TTL: 3600 (1 jam)

products:list:a1b2c3d4   ← hash dari query params (page, limit, category, sort)
→ JSON [{ ... }, { ... }]
  TTL: 900 (15 menit)
```

---

## Order Service

### Idempotency & Lock

| Key Pattern | Type | TTL | Value | Keterangan |
|---|---|---|---|---|
| `order:lock:{user_id}` | String | 10 detik | `"1"` | Mencegah double submit order |
| `order:idempotency:{key}` | String (JSON) | 24 jam | Response object | Idempotency key untuk create order |

---

## Payment Service

### Idempotency

| Key Pattern | Type | TTL | Value | Keterangan |
|---|---|---|---|---|
| `payment:idempotency:{key}` | String (JSON) | 24 jam | Response object | Mencegah double charge |
| `payment:pending:{order_id}` | String | 24 jam | `{payment_id}` | Tracking payment yang sedang pending |

---

## Gateway Service

### Rate Limiting

| Key Pattern | Type | TTL | Value | Keterangan |
|---|---|---|---|---|
| `rate:{ip}:{endpoint}` | Counter | 1 menit | `{count}` | Rate limit per IP per endpoint |
| `rate:user:{user_id}:{endpoint}` | Counter | 1 menit | `{count}` | Rate limit per user per endpoint |

**Limit defaults:**
```
Public endpoints    : 60 req/menit per IP
Auth endpoints      : 10 req/menit per IP
Authenticated user  : 120 req/menit per user
```

---

## Naming Convention

```
{service_domain}:{entity}:{identifier}
```

- Gunakan **snake_case**
- Pisahkan dengan titik dua `:`
- Jangan simpan data sensitif (password, full token) tanpa enkripsi
- Selalu set TTL — tidak ada key tanpa expiry kecuali ada alasan kuat

---

## Contoh Implementasi Go

```go
// Set cart item
key := fmt.Sprintf("cart:user:%s", userID)
err := rdb.HSet(ctx, key, variantID, quantity).Err()
rdb.Expire(ctx, key, 30*24*time.Hour)

// Get session
key := fmt.Sprintf("session:%s", userID)
data, err := rdb.HGetAll(ctx, key).Result()

// Rate limiting
key := fmt.Sprintf("rate:%s:%s", ip, endpoint)
count, err := rdb.Incr(ctx, key).Result()
if count == 1 {
    rdb.Expire(ctx, key, time.Minute)
}
if count > 60 {
    return ErrRateLimitExceeded
}
```