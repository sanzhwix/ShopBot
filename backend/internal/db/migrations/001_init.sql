-- All times are unix seconds (UTC). Seeded demo rows have is_demo = 1.

CREATE TABLE users (
    id           INTEGER PRIMARY KEY,           -- Telegram user id (demo customers are negative)
    first_name   TEXT    NOT NULL DEFAULT '',
    last_name    TEXT    NOT NULL DEFAULT '',
    username     TEXT    NOT NULL DEFAULT '',
    lang         TEXT    NOT NULL DEFAULT 'en',
    phone        TEXT    NOT NULL DEFAULT '',
    is_demo      INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL
);

CREATE TABLE malls (
    id         INTEGER PRIMARY KEY,
    slug       TEXT NOT NULL UNIQUE,
    name_en    TEXT NOT NULL,
    name_ru    TEXT NOT NULL,
    address_en TEXT NOT NULL,
    address_ru TEXT NOT NULL,
    lat        REAL NOT NULL,
    lng        REAL NOT NULL
);

CREATE TABLE shops (
    id             INTEGER PRIMARY KEY,
    slug           TEXT    NOT NULL UNIQUE,
    name           TEXT    NOT NULL,
    category       TEXT    NOT NULL,             -- clothes, bakery, grocery, pharmacy, electronics, flowers
    mall_id        INTEGER REFERENCES malls(id),
    lat            REAL    NOT NULL,
    lng            REAL    NOT NULL,
    address_en     TEXT    NOT NULL,
    address_ru     TEXT    NOT NULL,
    description_en TEXT    NOT NULL,
    description_ru TEXT    NOT NULL,
    hours          TEXT    NOT NULL,
    rating         REAL    NOT NULL DEFAULT 4.7,
    logo           TEXT    NOT NULL,             -- emoji
    kaspi_pay_url  TEXT    NOT NULL DEFAULT '',
    notify_chat_id INTEGER,                      -- Telegram chat that receives new orders
    created_at     INTEGER NOT NULL
);

CREATE TABLE shop_owners (
    id            INTEGER PRIMARY KEY,
    shop_id       INTEGER NOT NULL REFERENCES shops(id),
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    name          TEXT    NOT NULL,
    telegram_id   INTEGER UNIQUE
);

CREATE TABLE product_categories (
    id      INTEGER PRIMARY KEY,
    shop_id INTEGER NOT NULL REFERENCES shops(id),
    name_en TEXT    NOT NULL,
    name_ru TEXT    NOT NULL,
    sort    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE products (
    id             INTEGER PRIMARY KEY,
    shop_id        INTEGER NOT NULL REFERENCES shops(id),
    category_id    INTEGER NOT NULL REFERENCES product_categories(id),
    name_en        TEXT    NOT NULL,
    name_ru        TEXT    NOT NULL,
    description_en TEXT    NOT NULL DEFAULT '',
    description_ru TEXT    NOT NULL DEFAULT '',
    price          INTEGER NOT NULL,             -- tenge
    image          TEXT    NOT NULL,             -- emoji
    sizes          TEXT    NOT NULL DEFAULT '[]',-- JSON array
    colors         TEXT    NOT NULL DEFAULT '[]',-- JSON array of color codes
    material       TEXT    NOT NULL DEFAULT '',
    in_stock       INTEGER NOT NULL DEFAULT 1,
    created_at     INTEGER NOT NULL
);
CREATE INDEX idx_products_shop ON products(shop_id);

-- A customer's account "on the shop's website". Shops are hosted by ShopBot in the MVP.
CREATE TABLE shop_accounts (
    id            INTEGER PRIMARY KEY,
    shop_id       INTEGER NOT NULL REFERENCES shops(id),
    login         TEXT    NOT NULL,
    password_hash TEXT    NOT NULL,
    name          TEXT    NOT NULL DEFAULT '',
    telegram_id   INTEGER,
    created_at    INTEGER NOT NULL,
    UNIQUE (shop_id, login)
);

CREATE TABLE shop_sessions (
    user_id    INTEGER NOT NULL REFERENCES users(id),
    shop_id    INTEGER NOT NULL REFERENCES shops(id),
    account_id INTEGER NOT NULL REFERENCES shop_accounts(id),
    expires_at INTEGER NOT NULL,
    PRIMARY KEY (user_id, shop_id)
);

-- Saved shop logins ("My shop accounts"). secret is AES-256-GCM encrypted.
CREATE TABLE vault_entries (
    id           INTEGER PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    shop_id      INTEGER NOT NULL REFERENCES shops(id),
    login        TEXT    NOT NULL,
    secret       BLOB    NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER,
    UNIQUE (user_id, shop_id)
);

CREATE TABLE cart_items (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    product_id INTEGER NOT NULL REFERENCES products(id),
    size       TEXT    NOT NULL DEFAULT '',
    color      TEXT    NOT NULL DEFAULT '',
    qty        INTEGER NOT NULL,
    source     TEXT    NOT NULL DEFAULT 'direct',
    created_at INTEGER NOT NULL,
    UNIQUE (user_id, product_id, size, color)
);

CREATE TABLE orders (
    id             INTEGER PRIMARY KEY,
    shop_id        INTEGER NOT NULL REFERENCES shops(id),
    user_id        INTEGER NOT NULL REFERENCES users(id),
    account_id     INTEGER REFERENCES shop_accounts(id),
    status         TEXT    NOT NULL,             -- pending_payment, paid, accepted, ready, delivered, cancelled
    total          INTEGER NOT NULL,
    delivery_type  TEXT    NOT NULL,             -- pickup, delivery
    address        TEXT    NOT NULL DEFAULT '',
    phone          TEXT    NOT NULL DEFAULT '',
    comment        TEXT    NOT NULL DEFAULT '',
    source         TEXT    NOT NULL DEFAULT 'direct', -- map, search, direct
    payment_method TEXT    NOT NULL DEFAULT 'kaspi',
    is_demo        INTEGER NOT NULL DEFAULT 0,
    created_at     INTEGER NOT NULL,
    paid_at        INTEGER,
    updated_at     INTEGER NOT NULL
);
CREATE INDEX idx_orders_shop_time ON orders(shop_id, created_at);
CREATE INDEX idx_orders_user ON orders(user_id);

CREATE TABLE order_items (
    id         INTEGER PRIMARY KEY,
    order_id   INTEGER NOT NULL REFERENCES orders(id),
    product_id INTEGER NOT NULL REFERENCES products(id),
    name_en    TEXT    NOT NULL,
    name_ru    TEXT    NOT NULL,
    size       TEXT    NOT NULL DEFAULT '',
    color      TEXT    NOT NULL DEFAULT '',
    qty        INTEGER NOT NULL,
    price      INTEGER NOT NULL
);
CREATE INDEX idx_order_items_order ON order_items(order_id);

-- Telegram message that was sent to the shop for an order, so we can edit it later.
CREATE TABLE owner_messages (
    order_id   INTEGER NOT NULL REFERENCES orders(id),
    chat_id    INTEGER NOT NULL,
    message_id INTEGER NOT NULL,
    PRIMARY KEY (order_id, chat_id)
);

-- Real events from testers.
CREATE TABLE events (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL,
    type       TEXT    NOT NULL,
    shop_id    INTEGER,
    product_id INTEGER,
    source     TEXT    NOT NULL DEFAULT '',
    meta       TEXT    NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);
CREATE INDEX idx_events_shop_time ON events(shop_id, created_at);
CREATE INDEX idx_events_time ON events(created_at);

-- Seeded (imaginary) funnel history per shop per day. day = unix time of local midnight.
CREATE TABLE daily_stats (
    shop_id       INTEGER NOT NULL REFERENCES shops(id),
    day           INTEGER NOT NULL,
    pin_taps      INTEGER NOT NULL,
    shop_views    INTEGER NOT NULL,
    product_views INTEGER NOT NULL,
    add_to_carts  INTEGER NOT NULL,
    checkouts     INTEGER NOT NULL,
    PRIMARY KEY (shop_id, day)
);

-- Real shops from OpenStreetMap that a user asked to join ShopBot.
CREATE TABLE invites (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL,
    osm_id     TEXT    NOT NULL,
    name       TEXT    NOT NULL,
    category   TEXT    NOT NULL,
    lat        REAL    NOT NULL,
    lng        REAL    NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE (user_id, osm_id)
);

CREATE TABLE feedback (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL,
    rating     INTEGER NOT NULL,
    comment    TEXT    NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

CREATE TABLE osm_cache (
    key        TEXT PRIMARY KEY,
    body       TEXT    NOT NULL,
    fetched_at INTEGER NOT NULL
);
