-- =============================================================
-- Migration: 000002_add_verification_tokens (DOWN)
-- =============================================================

DROP INDEX IF EXISTS idx_pwd_reset_expires;
DROP INDEX IF EXISTS idx_pwd_reset_hash;
DROP INDEX IF EXISTS idx_pwd_reset_user_id;

DROP INDEX IF EXISTS idx_email_verif_expires;
DROP INDEX IF EXISTS idx_email_verif_hash;
DROP INDEX IF EXISTS idx_email_verif_user_id;

DROP TABLE IF EXISTS "password_reset_tokens";
DROP TABLE IF EXISTS "email_verification_tokens";
