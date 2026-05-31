-- =============================================================
-- infra/docker/postgres/init.sql
-- Dijalankan otomatis saat container Postgres pertama kali start
-- Membuat semua database untuk setiap service
-- =============================================================

CREATE DATABASE auth_db;
CREATE DATABASE product_db;
CREATE DATABASE order_db;
CREATE DATABASE payment_db;
CREATE DATABASE inventory_db;
CREATE DATABASE shipping_db;
CREATE DATABASE cart_db;
CREATE DATABASE notification_db;
CREATE DATABASE gateway_db;

-- Grant semua ke user default
GRANT ALL PRIVILEGES ON DATABASE auth_db         TO postgres;
GRANT ALL PRIVILEGES ON DATABASE product_db      TO postgres;
GRANT ALL PRIVILEGES ON DATABASE order_db        TO postgres;
GRANT ALL PRIVILEGES ON DATABASE payment_db      TO postgres;
GRANT ALL PRIVILEGES ON DATABASE inventory_db    TO postgres;
GRANT ALL PRIVILEGES ON DATABASE shipping_db     TO postgres;
GRANT ALL PRIVILEGES ON DATABASE cart_db         TO postgres;
GRANT ALL PRIVILEGES ON DATABASE notification_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE gateway_db      TO postgres;