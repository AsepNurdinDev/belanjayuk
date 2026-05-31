# =============================================================
# Makefile — Migration Targets
# Tambahkan bagian ini ke Makefile root project kamu
# =============================================================
#
# Cara pakai:
#   make migrate-up SERVICE=auth-service
#   make migrate-down SERVICE=auth-service
#   make migrate-up-all
#   make migrate-create SERVICE=auth-service NAME=add_sessions
# =============================================================

# Load .env dari service yang dipilih (untuk DB credentials)
# Jika pakai Docker Compose, DB_URL bisa di-override dari luar
DB_HOST     ?= localhost
DB_PORT     ?= 5432
DB_USER     ?= postgres
DB_PASSWORD ?= postgres
DB_SSL_MODE ?= disable

# Mapping service → nama database
db_name_auth-service         = auth_db
db_name_product-service      = product_db
db_name_order-service        = order_db
db_name_payment-service      = payment_db
db_name_inventory-service    = inventory_db
db_name_shipping-service     = shipping_db
db_name_notification-service = notification_db
db_name_analytics-service    = analytics_db
db_name_cart-service         = cart_db
db_name_gateway-service      = gateway_db

# Build DB URL berdasarkan SERVICE
define get_db_url
postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(db_name_$(1))?sslmode=$(DB_SSL_MODE)
endef

# -------------------------------------------------------------
# migrate-up SERVICE=<nama-service>
# Jalankan semua pending migration untuk 1 service
# -------------------------------------------------------------
.PHONY: migrate-up
migrate-up:
ifndef SERVICE
	$(error SERVICE is required. Usage: make migrate-up SERVICE=auth-service)
endif
	@echo "▶ Running migrations UP for $(SERVICE)..."
	@migrate \
		-path ./services/$(SERVICE)/migrations \
		-database "$(call get_db_url,$(SERVICE))" \
		up
	@echo "✓ Migration UP done for $(SERVICE)"

# -------------------------------------------------------------
# migrate-down SERVICE=<nama-service>
# Rollback 1 step migration
# -------------------------------------------------------------
.PHONY: migrate-down
migrate-down:
ifndef SERVICE
	$(error SERVICE is required. Usage: make migrate-down SERVICE=auth-service)
endif
	@echo "▶ Running migrations DOWN for $(SERVICE)..."
	@migrate \
		-path ./services/$(SERVICE)/migrations \
		-database "$(call get_db_url,$(SERVICE))" \
		down 1
	@echo "✓ Migration DOWN done for $(SERVICE)"

# -------------------------------------------------------------
# migrate-up-all
# Jalankan migration semua service sekaligus
# -------------------------------------------------------------
SERVICES := auth-service product-service order-service payment-service \
            inventory-service shipping-service cart-service

.PHONY: migrate-up-all
migrate-up-all:
	@echo "▶ Running migrations UP for all services..."
	@for svc in $(SERVICES); do \
		echo "  → $$svc"; \
		migrate \
			-path ./services/$$svc/migrations \
			-database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$$(make --no-print-directory _db-name-$$svc)?sslmode=$(DB_SSL_MODE)" \
			up || exit 1; \
	done
	@echo "✓ All migrations done"

# -------------------------------------------------------------
# migrate-create SERVICE=<nama-service> NAME=<nama_migration>
# Buat file migration baru (up + down)
# -------------------------------------------------------------
.PHONY: migrate-create
migrate-create:
ifndef SERVICE
	$(error SERVICE is required. Usage: make migrate-create SERVICE=auth-service NAME=add_sessions)
endif
ifndef NAME
	$(error NAME is required. Usage: make migrate-create SERVICE=auth-service NAME=add_sessions)
endif
	@migrate create \
		-ext sql \
		-dir ./services/$(SERVICE)/migrations \
		-seq $(NAME)
	@echo "✓ Created migration files for $(NAME) in $(SERVICE)"

# -------------------------------------------------------------
# migrate-version SERVICE=<nama-service>
# Cek versi migration yang sedang aktif
# -------------------------------------------------------------
.PHONY: migrate-version
migrate-version:
ifndef SERVICE
	$(error SERVICE is required. Usage: make migrate-version SERVICE=auth-service)
endif
	@migrate \
		-path ./services/$(SERVICE)/migrations \
		-database "$(call get_db_url,$(SERVICE))" \
		version

# -------------------------------------------------------------
# migrate-force SERVICE=<nama-service> VERSION=<nomor>
# Force set versi (untuk fix dirty state)
# -------------------------------------------------------------
.PHONY: migrate-force
migrate-force:
ifndef SERVICE
	$(error SERVICE is required)
endif
ifndef VERSION
	$(error VERSION is required. Usage: make migrate-force SERVICE=auth-service VERSION=1)
endif
	@migrate \
		-path ./services/$(SERVICE)/migrations \
		-database "$(call get_db_url,$(SERVICE))" \
		force $(VERSION)
	@echo "✓ Forced version $(VERSION) for $(SERVICE)"