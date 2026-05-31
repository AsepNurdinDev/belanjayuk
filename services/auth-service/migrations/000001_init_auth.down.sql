-- =============================================================
-- Migration: 000001_init_auth (DOWN)
-- Drop dalam urutan terbalik karena ada foreign key
-- =============================================================

DROP INDEX IF EXISTS idx_refresh_tokens_expires;
DROP INDEX IF EXISTS idx_refresh_tokens_hash;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;

DROP INDEX IF EXISTS idx_user_addresses_default;
DROP INDEX IF EXISTS idx_user_addresses_user_id;

DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_google_id;
DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS "refresh_tokens";
DROP TABLE IF EXISTS "user_addresses";
DROP TABLE IF EXISTS "user_profiles";
DROP TABLE IF EXISTS "users";