CREATE TABLE "users" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "email" varchar(255) UNIQUE NOT NULL,
  "password_hash" varchar(255) NOT NULL,
  "full_name" varchar(100) NOT NULL,
  "phone" varchar(20),
  "is_verified" boolean DEFAULT false,
  "auth_method" varchar(20) DEFAULT 'local',
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now())
);

CREATE TABLE "user_profiles" (
  "id"          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "user_id"     uuid UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  "avatar_url"  varchar(500),
  "bio"         text,
  "birth_date"  date,
  "gender"      varchar(10),
  "created_at"  timestamp DEFAULT now(),
  "updated_at"  timestamp DEFAULT now()
);

CREATE TABLE "user_addresses" (
  "id"           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "user_id"      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  "label"        varchar(50) NOT NULL,  -- 'Rumah', 'Kantor', dll
  "recipient"    varchar(100) NOT NULL,
  "phone"        varchar(20) NOT NULL,
  "province"     varchar(100) NOT NULL,
  "city"         varchar(100) NOT NULL,
  "district"     varchar(100) NOT NULL,
  "postal_code"  varchar(10) NOT NULL,
  "detail"       text NOT NULL,
  "is_default"   boolean DEFAULT false,
  "created_at"   timestamp DEFAULT now(),
  "updated_at"   timestamp DEFAULT now()
);

CREATE TABLE "categories" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "parent_id" uuid,
  "name" varchar(100) NOT NULL,
  "slug" varchar(100) UNIQUE NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now())
);

CREATE TABLE "products" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "category_id" uuid,
  "name" varchar(255) NOT NULL,
  "slug" varchar(255) UNIQUE NOT NULL,
  "description" text,
  "is_active" boolean DEFAULT true,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now()),
  "deleted_at" timestamp
);

CREATE TABLE "product_images" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "product_id" uuid,
  "image_url" varchar(255) NOT NULL,
  "is_primary" boolean DEFAULT false,
  "created_at" timestamp DEFAULT (now())
);

CREATE TABLE "product_variants" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "product_id" uuid,
  "sku" varchar(100) UNIQUE NOT NULL,
  "name" varchar(100) NOT NULL,
  "price" numeric(15,2) NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now())
);

CREATE TABLE "inventories" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "variant_id" uuid UNIQUE NOT NULL,
  "stock_quantity" integer NOT NULL DEFAULT 0,
  "reserved_quantity" integer NOT NULL DEFAULT 0,
  "low_stock_threshold" integer DEFAULT 5,
  "updated_at" timestamp DEFAULT (now())
);

CREATE TABLE "orders" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid NOT NULL,
  "order_number" varchar(50) UNIQUE NOT NULL,
  "status" varchar(20) NOT NULL,
  "total_amount" numeric(15,2) NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now())
);

CREATE TABLE "order_items" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "order_id" uuid,
  "variant_id" uuid NOT NULL,
  "quantity" integer NOT NULL,
  "price_at_purchase" numeric(15,2) NOT NULL,
  "created_at" timestamp DEFAULT (now())
);

CREATE TABLE "payments" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "order_id" uuid UNIQUE NOT NULL,
  "transaction_id" varchar(255) UNIQUE,
  "payment_gateway" varchar(50) NOT NULL,
  "payment_method" varchar(50),
  "status" varchar(20) NOT NULL,
  "amount" numeric(15,2) NOT NULL,
  "idempotency_key" varchar(255) UNIQUE NOT NULL,
  "paid_at" timestamp,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now())
);

CREATE TABLE "shippings" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "order_id" uuid UNIQUE NOT NULL,
  "courier" varchar(50) NOT NULL,
  "service" varchar(50) NOT NULL,
  "tracking_number" varchar(100) UNIQUE,
  "status" varchar(50) NOT NULL,
  "shipping_cost" numeric(15,2) NOT NULL,
  "address_details" text NOT NULL,
  "created_at" timestamp DEFAULT (now()),
  "updated_at" timestamp DEFAULT (now())
);

COMMENT ON COLUMN "users"."auth_method" IS 'local, google, github';

COMMENT ON COLUMN "product_variants"."name" IS 'e.g., Red XL, Blue L';

COMMENT ON COLUMN "inventories"."variant_id" IS 'References product_variants.id via microservice communication';

COMMENT ON COLUMN "inventories"."reserved_quantity" IS 'Held temporarily during checkout';

COMMENT ON COLUMN "orders"."user_id" IS 'References users.id';

COMMENT ON COLUMN "orders"."order_number" IS 'e.g., BJ-20260601-00001';

COMMENT ON COLUMN "orders"."status" IS 'pending, confirmed, processing, shipped, delivered, completed, cancelled';

COMMENT ON COLUMN "order_items"."variant_id" IS 'References product_variants.id';

COMMENT ON COLUMN "payments"."order_id" IS 'References orders.id';

COMMENT ON COLUMN "payments"."transaction_id" IS 'Gateway transaction reference ID';

COMMENT ON COLUMN "payments"."payment_gateway" IS 'midtrans, xendit';

COMMENT ON COLUMN "payments"."payment_method" IS 'bank_transfer, ewallet, credit_card';

COMMENT ON COLUMN "payments"."status" IS 'pending, success, failed, expired, refunded';

COMMENT ON COLUMN "shippings"."order_id" IS 'References orders.id';

COMMENT ON COLUMN "shippings"."courier" IS 'jne, jnt, sicepat';

COMMENT ON COLUMN "shippings"."service" IS 'reg, yes, oke';

COMMENT ON COLUMN "shippings"."status" IS 'pending, pickup, in_transit, delivered';

ALTER TABLE "categories" ADD FOREIGN KEY ("parent_id") REFERENCES "categories" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "products" ADD FOREIGN KEY ("category_id") REFERENCES "categories" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "product_images" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "product_variants" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "order_items" ADD FOREIGN KEY ("order_id") REFERENCES "orders" ("id") DEFERRABLE INITIALLY IMMEDIATE;
