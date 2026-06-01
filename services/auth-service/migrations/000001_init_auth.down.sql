-- =============================================================
-- Migration: 000001_init_auth (DOWN)
-- Rollback semua tabel auth service
-- =============================================================

-- Hapus indexes dulu (otomatis terhapus dengan DROP TABLE, tapi eksplisit lebih aman)
DROP INDEX IF EXISTS idx_refresh_tokens_expires;
DROP INDEX IF EXISTS idx_refresh_tokens_hash;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;

DROP INDEX IF EXISTS idx_user_addresses_default;
DROP INDEX IF EXISTS idx_user_addresses_user_id;

DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_google_id;
DROP INDEX IF EXISTS idx_users_email;

-- Hapus tabel (urutan terbalik dari CREATE karena foreign key)
DROP TABLE IF EXISTS "refresh_tokens";
DROP TABLE IF EXISTS "user_addresses";
DROP TABLE IF EXISTS "user_profiles";
DROP TABLE IF EXISTS "users";
