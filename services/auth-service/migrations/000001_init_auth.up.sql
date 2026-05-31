-- =============================================================
-- Migration: 000001_init_auth (UP)
-- Auth Service: users, user_profiles, user_addresses
-- =============================================================

-- -------------------------------------------------------------
-- USERS
-- Core authentication data only
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS "users" (
    "id"            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "email"         VARCHAR(255) UNIQUE NOT NULL,
    "password_hash" VARCHAR(255),                           -- nullable: Google user tidak punya password
    "full_name"     VARCHAR(100) NOT NULL,
    "phone"         VARCHAR(20),
    "role"          VARCHAR(20) NOT NULL DEFAULT 'customer', -- customer | admin | super_admin
    "auth_method"   VARCHAR(20) NOT NULL DEFAULT 'local',   -- local | google
    "google_id"     VARCHAR(100) UNIQUE,                    -- nullable: hanya Google user
    "is_verified"   BOOLEAN NOT NULL DEFAULT FALSE,
    "created_at"    TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at"    TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN "users"."password_hash" IS 'NULL untuk user yang register via Google OAuth';
COMMENT ON COLUMN "users"."role"          IS 'customer, admin, super_admin';
COMMENT ON COLUMN "users"."auth_method"   IS 'local, google';
COMMENT ON COLUMN "users"."google_id"     IS 'Sub ID dari Google OAuth, NULL untuk local user';

-- -------------------------------------------------------------
-- USER PROFILES
-- Extended identity data, dipisah dari users agar auth-table tetap lean
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS "user_profiles" (
    "id"          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id"     UUID UNIQUE NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
    "avatar_url"  VARCHAR(500),
    "bio"         TEXT,
    "birth_date"  DATE,
    "gender"      VARCHAR(10),                              -- male | female | other
    "created_at"  TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at"  TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN "user_profiles"."gender" IS 'male, female, other';

-- -------------------------------------------------------------
-- USER ADDRESSES
-- Alamat pengiriman milik user, di-snapshot ke shippings.address_details saat order
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS "user_addresses" (
    "id"          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id"     UUID NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
    "label"       VARCHAR(50) NOT NULL,                     -- Rumah, Kantor, dll
    "recipient"   VARCHAR(100) NOT NULL,
    "phone"       VARCHAR(20) NOT NULL,
    "province"    VARCHAR(100) NOT NULL,
    "city"        VARCHAR(100) NOT NULL,
    "district"    VARCHAR(100) NOT NULL,
    "postal_code" VARCHAR(10) NOT NULL,
    "detail"      TEXT NOT NULL,
    "is_default"  BOOLEAN NOT NULL DEFAULT FALSE,
    "created_at"  TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at"  TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN "user_addresses"."label"      IS 'Label bebas: Rumah, Kantor, Kos, dll';
COMMENT ON COLUMN "user_addresses"."is_default" IS 'Hanya 1 alamat default per user, di-enforce di aplikasi';

-- -------------------------------------------------------------
-- REFRESH TOKENS
-- Disimpan di Redis untuk produksi, tabel ini sebagai fallback / audit trail
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS "refresh_tokens" (
    "id"         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id"    UUID NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
    "token_hash" VARCHAR(255) UNIQUE NOT NULL,              -- bcrypt/sha256 dari raw token
    "expires_at" TIMESTAMP NOT NULL,
    "revoked_at" TIMESTAMP,                                 -- NULL = masih valid
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN "refresh_tokens"."token_hash" IS 'Hash dari refresh token, bukan raw value';
COMMENT ON COLUMN "refresh_tokens"."revoked_at" IS 'Di-set saat logout atau token di-rotate';

-- -------------------------------------------------------------
-- INDEXES
-- -------------------------------------------------------------
CREATE INDEX idx_users_email        ON "users" ("email");
CREATE INDEX idx_users_google_id    ON "users" ("google_id") WHERE "google_id" IS NOT NULL;
CREATE INDEX idx_users_role         ON "users" ("role");

CREATE INDEX idx_user_addresses_user_id   ON "user_addresses" ("user_id");
CREATE INDEX idx_user_addresses_default   ON "user_addresses" ("user_id", "is_default");

CREATE INDEX idx_refresh_tokens_user_id   ON "refresh_tokens" ("user_id");
CREATE INDEX idx_refresh_tokens_hash      ON "refresh_tokens" ("token_hash");
CREATE INDEX idx_refresh_tokens_expires   ON "refresh_tokens" ("expires_at");