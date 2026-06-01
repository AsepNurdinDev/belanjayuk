-- =============================================================
-- Migration: 000002_add_verification_tokens (UP)
-- Email verification dan password reset tokens
-- =============================================================

-- -------------------------------------------------------------
-- EMAIL VERIFICATION TOKENS
-- Digunakan untuk verifikasi email saat register (local user)
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS "email_verification_tokens" (
    "id"         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id"    UUID NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
    "token_hash" VARCHAR(255) UNIQUE NOT NULL,
    "expires_at" TIMESTAMP NOT NULL,
    "used_at"    TIMESTAMP,                -- NULL = belum digunakan
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE "email_verification_tokens" IS 'Token satu kali pakai untuk verifikasi email';
COMMENT ON COLUMN "email_verification_tokens"."used_at" IS 'Di-set saat token digunakan, tidak bisa dipakai ulang';

-- -------------------------------------------------------------
-- PASSWORD RESET TOKENS
-- Digunakan untuk alur "lupa password"
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS "password_reset_tokens" (
    "id"         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id"    UUID NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
    "token_hash" VARCHAR(255) UNIQUE NOT NULL,
    "expires_at" TIMESTAMP NOT NULL,
    "used_at"    TIMESTAMP,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE "password_reset_tokens" IS 'Token satu kali pakai untuk reset password';

-- Indexes
CREATE INDEX idx_email_verif_user_id  ON "email_verification_tokens" ("user_id");
CREATE INDEX idx_email_verif_hash     ON "email_verification_tokens" ("token_hash");
CREATE INDEX idx_email_verif_expires  ON "email_verification_tokens" ("expires_at");

CREATE INDEX idx_pwd_reset_user_id    ON "password_reset_tokens" ("user_id");
CREATE INDEX idx_pwd_reset_hash       ON "password_reset_tokens" ("token_hash");
CREATE INDEX idx_pwd_reset_expires    ON "password_reset_tokens" ("expires_at");
