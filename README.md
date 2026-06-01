# 🛒 Ecommerce Microservices Platform

Platform e-commerce berbasis microservices menggunakan **Go** untuk backend dan **Next.js** untuk frontend.

---

## 📋 Daftar Isi

- [Arsitektur](#arsitektur)
- [Tech Stack](#tech-stack)
- [Struktur Folder](#struktur-folder)
- [Services](#services)
- [Prerequisites](#prerequisites)
- [Cara Menjalankan](#cara-menjalankan)
- [Environment Variables](#environment-variables)
- [API Documentation](#api-documentation)
- [Testing](#testing)
- [Deployment](#deployment)
- [Monitoring](#monitoring)
- [Contributing](#contributing)

---

## Arsitektur

```
                        ┌─────────────┐   ┌──────────────┐
                        │ frontend-   │   │  frontend-   │
                        │   user      │   │   admin      │
                        │ (Next.js)   │   │  (Next.js)   │
                        └──────┬──────┘   └──────┬───────┘
                               │                 │
                               ▼                 ▼
                        ┌──────────────────────────┐
                        │       NGINX / Ingress     │
                        └──────────────┬───────────┘
                                       │
                                       ▼
                        ┌──────────────────────────┐
                        │      Gateway Service      │
                        │  (Auth, Rate Limit, CORS) │
                        └──────┬───────────────┬───┘
                               │               │
              ┌────────────────┼───────────────┼────────────────┐
              │                │               │                │
              ▼                ▼               ▼                ▼
       ┌────────────┐  ┌──────────────┐ ┌──────────┐  ┌──────────────┐
       │    Auth    │  │   Product    │ │   Cart   │  │    Order     │
       │  Service   │  │   Service   │ │  Service │  │   Service    │
       └────────────┘  └──────────────┘ └──────────┘  └──────┬───────┘
                                                              │
                              ┌───────────────────────────────┤
                              │               │               │
                              ▼               ▼               ▼
                       ┌──────────┐  ┌─────────────┐ ┌──────────────┐
                       │ Payment  │  │  Inventory  │ │   Shipping   │
                       │ Service  │  │   Service   │ │   Service    │
                       └──────────┘  └─────────────┘ └──────────────┘
                              │               │               │
                              └───────────────┴───────────────┘
                                              │
                                     ┌────────▼────────┐
                                     │    RabbitMQ     │
                                     │  (Event Bus)    │
                                     └────────┬────────┘
                                              │
                              ┌───────────────┴──────────────┐
                              │                              │
                              ▼                              ▼
                    ┌──────────────────┐          ┌──────────────────┐
                    │   Notification   │          │    Analytics     │
                    │     Service      │          │     Service      │
                    └──────────────────┘          └──────────────────┘
```

---

## Tech Stack

### Backend
| Komponen | Teknologi |
|----------|-----------|
| Language | Go 1.22+ |
| HTTP Framework | [Gin](https://github.com/gin-gonic/gin) / [Echo](https://echo.labstack.com/) |
| ORM / Query Builder | [sqlc](https://sqlc.dev/) + [pgx](https://github.com/jackc/pgx) |
| Config | [Viper](https://github.com/spf13/viper) |
| Logging | [zerolog](https://github.com/rs/zerolog) |
| Validation | [go-playground/validator](https://github.com/go-playground/validator) |
| JWT | [golang-jwt/jwt](https://github.com/golang-jwt/jwt) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Message Queue | RabbitMQ ([amqp091-go](https://github.com/rabbitmq/amqp091-go)) |
| gRPC | [google.golang.org/grpc](https://pkg.go.dev/google.golang.org/grpc) |
| API Docs | [swaggo/swag](https://github.com/swaggo/swag) |
| Hot Reload (dev) | [air](https://github.com/air-verse/air) |

### Frontend
| Komponen | Teknologi |
|----------|-----------|
| Framework | Next.js 14+ (App Router) |
| Language | TypeScript |
| Styling | Tailwind CSS |
| UI Components | shadcn/ui |
| State Management | Zustand |
| HTTP Client | Axios |
| Form | React Hook Form + Zod |
| Testing | Vitest + Playwright |

### Infrastructure
| Komponen | Teknologi |
|----------|-----------|
| Primary DB | PostgreSQL 16 |
| Cache / Session | Redis 7 |
| Document DB | MongoDB 7 |
| Message Broker | RabbitMQ 3 |
| Reverse Proxy | Nginx |
| Container | Docker + Docker Compose |
| Orchestration | Kubernetes (EKS/GKE) |
| IaC | Terraform |
| CI/CD | GitHub Actions + Jenkins |
| Monitoring | Prometheus + Grafana + Loki |

---

## Struktur Folder

```
ecommerce/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   ├── cd-staging.yml
│   │   └── cd-production.yml
│   └── PULL_REQUEST_TEMPLATE.md
│
├── services/                        # Semua Go microservices
│   ├── gateway-service/
│   ├── auth-service/
│   ├── product-service/
│   ├── cart-service/
│   ├── order-service/
│   ├── payment-service/
│   ├── inventory-service/
│   ├── shipping-service/
│   ├── notification-service/
│   └── analytics-service/
│
├── frontend/
│   ├── user/                        # Next.js storefront
│   └── admin/                       # Next.js dashboard admin
│
├── shared/                          # Go shared packages
│   ├── pkg/
│   │   ├── logger/
│   │   ├── middleware/
│   │   ├── errors/
│   │   ├── jwt/
│   │   ├── pagination/
│   │   └── validator/
│   └── proto/                       # Protobuf definitions
│       ├── auth.proto
│       ├── product.proto
│       └── order.proto
│
├── infra/
│   ├── terraform/
│   ├── k8s/
│   └── helm/
│
├── monitoring/
│   ├── prometheus/
│   ├── grafana/
│   └── loki/
│
├── docker/                          # Config infra lokal
│   ├── postgres/
│   ├── redis/
│   ├── mongodb/
│   └── rabbitmq/
│
├── scripts/
├── docker-compose.yml
├── docker-compose.dev.yml
├── go.work                          # Go workspace (multi-module)
├── Makefile
└── README.md
```

### Struktur Per Go Service

Setiap Go service mengikuti **Standard Go Project Layout** dengan pola **Clean Architecture**:

```
services/{nama}-service/
├── cmd/
│   └── api/
│       └── main.go              # Entry point, dependency injection
│
├── internal/                    # Kode private (tidak bisa di-import luar)
│   ├── config/
│   │   └── config.go            # Config via env / Viper
│   │
│   ├── domain/                  # Entities & business rules (paling dalam)
│   │   ├── entity.go            # Struct domain + domain errors
│   │   └── repository.go        # Interface repository
│   │
│   ├── usecase/                 # Business logic
│   │   ├── usecase.go
│   │   └── usecase_test.go
│   │
│   ├── repository/              # Implementasi interface domain
│   │   ├── postgres/
│   │   │   └── repo.go
│   │   └── redis/
│   │       └── repo.go
│   │
│   ├── delivery/                # Transport layer
│   │   ├── http/
│   │   │   ├── handler.go
│   │   │   ├── request.go       # DTO input
│   │   │   ├── response.go      # DTO output
│   │   │   └── router.go
│   │   └── grpc/
│   │       └── server.go
│   │
│   ├── middleware/
│   └── event/                   # RabbitMQ publisher & subscriber
│
├── pkg/                         # Public helpers (boleh di-import)
│   └── {service}client/
│       └── client.go            # gRPC client stub
│
├── migrations/                  # SQL (golang-migrate format)
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
│
├── docs/                        # Swagger (auto-generated swaggo)
├── .env.example
├── Dockerfile
├── Dockerfile.dev               # dengan air hot-reload
├── go.mod
└── go.sum
```

### Dependency Rule (Clean Architecture)

```
delivery → usecase → domain ← repository
   ↑                              ↑
(HTTP/gRPC)                  (Postgres/Redis)
```

- `domain` tidak bergantung pada siapapun
- `usecase` hanya tahu `domain`
- `delivery` dan `repository` bergantung pada `domain`

---

## Services

### Gateway Service — `:8000`
Pintu masuk tunggal semua request. Bertanggung jawab atas:
- JWT verification sebelum forward ke downstream
- Rate limiting per IP / per user
- CORS handling
- Request ID injection untuk distributed tracing
- Circuit breaker ke downstream service

### Auth Service — `:8001`
- Register, login, logout
- JWT access token + refresh token (disimpan di Redis)
- OAuth2 (Google, GitHub)
- Password reset via email

### Product Service — `:8002`
- CRUD produk dengan kategori dan tag
- Upload gambar produk (S3 / MinIO)
- Search & filter (Elasticsearch / PostgreSQL full-text)
- Soft delete

### Cart Service — `:8003`
- Cart tersimpan di Redis (guest & logged-in user)
- Merge cart saat user login
- Validasi stok saat add to cart

### Order Service — `:8004`
- Buat order dari cart
- Order state machine: `pending → confirmed → processing → shipped → delivered → completed`
- Integrasi dengan inventory, payment, dan shipping service via events

### Payment Service — `:8005`
- Integrasi Midtrans & Xendit
- Webhook handler dengan HMAC verification
- Idempotency key untuk mencegah double charge
- Refund management

### Inventory Service — `:8006`
- Manajemen stok per SKU
- Stock reservation (hold saat checkout)
- Stock release saat order cancelled / expired
- Low stock alert via event

### Shipping Service — `:8007`
- Integrasi RajaOngkir / Shipper
- Kalkulasi ongkos kirim
- Tracking status pengiriman
- Webhook dari kurir

### Notification Service — `:8008`
- Murni event-driven, tidak expose HTTP endpoint publik
- Multi-channel: Email (SendGrid), SMS (Twilio), Push (FCM)
- Template-based dengan `html/template`

### Analytics Service — `:8009`
- Consume semua business events dari RabbitMQ
- Tulis ke MongoDB untuk analitik
- Expose endpoint untuk dashboard admin

---

## Prerequisites

Pastikan sudah terinstall:

- [Go](https://golang.org/dl/) 1.22+
- [Node.js](https://nodejs.org/) 20+ & npm/pnpm
- [Docker](https://www.docker.com/) & Docker Compose v2
- [Make](https://www.gnu.org/software/make/)
- [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)
- [air](https://github.com/air-verse/air) (hot reload)
- [swag CLI](https://github.com/swaggo/swag) (API docs)
- [protoc](https://grpc.io/docs/protoc-installation/) + Go plugins (jika pakai gRPC)

```bash
# Install Go tools
go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/cmd/swag@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install protoc Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

---

## Cara Menjalankan

### 1. Clone & Setup

```bash
git clone https://github.com/your-org/ecommerce.git
cd ecommerce

# Copy semua .env.example
make env-setup

# Atau manual per service:
cp services/auth-service/.env.example services/auth-service/.env
# ... ulangi untuk setiap service
```

### 2. Jalankan Infrastruktur (DB, Redis, RabbitMQ)

```bash
# Jalankan hanya infrastruktur
docker compose up postgres redis mongodb rabbitmq -d

# Cek status
docker compose ps
```

### 3. Jalankan Migrasi

```bash
# Semua service sekaligus
make migrate-up

# Atau per service
make migrate-up SERVICE=auth-service
```

### 4. Seed Data (Development)

```bash
make seed
```

### 5. Jalankan Semua Services

```bash
# Mode development (dengan hot-reload via air)
make dev

# Atau pakai Docker Compose penuh
docker compose -f docker-compose.dev.yml up --build
```

### 6. Jalankan Frontend

```bash
# User storefront
cd frontend/user
npm install
npm run dev         # http://localhost:3000

# Admin dashboard
cd frontend/admin
npm install
npm run dev         # http://localhost:3001
```

### Makefile Targets

```bash
make dev            # Jalankan semua services (hot-reload)
make build          # Build semua Docker images
make test           # Jalankan semua unit + integration tests
make lint           # golangci-lint semua services
make migrate-up     # Jalankan semua migrasi
make migrate-down   # Rollback semua migrasi
make gen-proto      # Generate kode dari .proto files
make gen-swagger    # Generate Swagger docs semua services
make seed           # Seed database development
make clean          # Hapus build artifacts
make help           # Tampilkan semua target
```

---

## Environment Variables

Setiap service memiliki `.env.example`. Berikut variabel utama:

### Shared (semua service)
```env
APP_ENV=development          # development | staging | production
APP_PORT=8001
LOG_LEVEL=debug              # debug | info | warn | error

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=auth_db
DB_USER=postgres
DB_PASSWORD=secret
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

### Auth Service
```env
JWT_ACCESS_SECRET=your-access-secret-here
JWT_REFRESH_SECRET=your-refresh-secret-here
JWT_ACCESS_EXPIRE=15m
JWT_REFRESH_EXPIRE=7d

GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

### Payment Service
```env
MIDTRANS_SERVER_KEY=
MIDTRANS_CLIENT_KEY=
MIDTRANS_ENV=sandbox           # sandbox | production
XENDIT_SECRET_KEY=
XENDIT_WEBHOOK_TOKEN=
```

### Notification Service
```env
SENDGRID_API_KEY=
EMAIL_FROM=noreply@yourdomain.com
TWILIO_ACCOUNT_SID=
TWILIO_AUTH_TOKEN=
TWILIO_PHONE_NUMBER=
FIREBASE_CREDENTIALS_PATH=./firebase-credentials.json
```

---

## API Documentation

Swagger UI tersedia saat service berjalan:

| Service | URL |
|---------|-----|
| Gateway | http://localhost:8000/swagger/index.html |
| Auth | http://localhost:8001/swagger/index.html |
| Product | http://localhost:8002/swagger/index.html |
| Order | http://localhost:8004/swagger/index.html |
| Payment | http://localhost:8005/swagger/index.html |

Generate ulang docs setelah mengubah komentar swaggo:

```bash
make gen-swagger

# Atau per service
cd services/auth-service && swag init -g cmd/api/main.go
```

---

## Testing

### Unit Tests

```bash
# Semua service
make test

# Per service
cd services/auth-service
go test ./internal/usecase/... -v -cover

# Dengan coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Jalankan infra test dulu
docker compose -f docker-compose.test.yml up -d

# Jalankan integration tests
make test-integration

# Cleanup
docker compose -f docker-compose.test.yml down
```

### E2E Tests (Frontend)

```bash
cd frontend/user
npx playwright test

# UI mode
npx playwright test --ui
```

### Load Testing

```bash
# Menggunakan k6
k6 run scripts/load-test/checkout.js
```

---

## Deployment

### Staging (otomatis)

Push ke branch `main` akan trigger GitHub Actions untuk deploy ke staging:

```
push to main → CI (lint + test) → build & push Docker image → deploy to staging k8s
```

### Production (manual approval)

```bash
# Via GitHub Actions — perlu approval dari maintainer
# Atau manual:
make deploy-production VERSION=v1.2.3
```

### Build Docker Image

```bash
# Build satu service
docker build -t ecommerce/auth-service:latest ./services/auth-service

# Build semua
make build
```

### Kubernetes

```bash
# Apply ke staging
kubectl apply -k infra/k8s/overlays/staging

# Apply ke production
kubectl apply -k infra/k8s/overlays/production

# Cek status pods
kubectl get pods -n ecommerce
```

---

## Monitoring

| Tool | URL (local) | Keterangan |
|------|-------------|------------|
| Grafana | http://localhost:3100 | Dashboard metrics & logs |
| Prometheus | http://localhost:9090 | Metrics scraping |
| Loki | http://localhost:3200 | Log aggregation |
| RabbitMQ Management | http://localhost:15672 | Queue monitoring |

### Jalankan Stack Monitoring

```bash
docker compose up prometheus grafana loki -d
```

Grafana default credentials: `admin / admin`

Dashboard yang tersedia:
- **Overview** — health semua services
- **Go Runtime** — goroutines, GC, memory heap per service
- **Services** — request rate, latency, error rate (RED metrics)
- **Database** — query performance, connection pool
- **Business Metrics** — GMV, order count, conversion rate

---

## Event Flow (RabbitMQ)

| Event | Publisher | Subscriber(s) |
|-------|-----------|---------------|
| `user.registered` | auth-service | notification-service |
| `order.created` | order-service | payment-service, inventory-service |
| `payment.success` | payment-service | order-service, notification-service, analytics-service |
| `payment.failed` | payment-service | order-service, notification-service |
| `inventory.reserved` | inventory-service | order-service |
| `inventory.low_stock` | inventory-service | notification-service |
| `shipping.updated` | shipping-service | order-service, notification-service |

---

## Contributing

1. Fork repository ini
2. Buat branch fitur: `git checkout -b feat/nama-fitur`
3. Commit dengan [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat(auth): add Google OAuth2 login
   fix(payment): handle duplicate webhook event
   chore(deps): update gin to v1.10
   ```
4. Pastikan semua test lulus: `make test`
5. Pastikan lint bersih: `make lint`
6. Buat Pull Request ke `main`

### Code Style

- Go: ikuti [Effective Go](https://go.dev/doc/effective_go) + `golangci-lint`
- TypeScript: ESLint + Prettier (konfigurasi sudah ada di root)
- Commit message: Conventional Commits

---

## License

MIT License — lihat file [LICENSE](LICENSE) untuk detail.