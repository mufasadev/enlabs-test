BEGIN;

SET client_encoding = 'UTF8';
SET standard_conforming_strings = ON;
SET default_with_oids = FALSE;
SET search_path = public, extensions;
SET check_function_bodies = FALSE;
SET client_min_messages = WARNING;

-- EXTENSIONS --
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- TABLES --
CREATE TABLE IF NOT EXISTS users
(
    id             UUID PRIMARY KEY        DEFAULT gen_random_uuid(),
    balance        NUMERIC(10, 2) NOT NULL DEFAULT 0,
    account_number UUID           NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS sources
(
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(20) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS transactions
(
    id             UUID PRIMARY KEY        DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(255)    NOT NULL UNIQUE,
    state          VARCHAR(4)     NOT NULL,
    amount         NUMERIC(10, 2) NOT NULL,
    source_id      UUID           NOT NULL REFERENCES sources (id),
    processed      BOOLEAN        NOT NULL DEFAULT FALSE,
    user_id        UUID           NOT NULL REFERENCES users (id),
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- INDEXES --
CREATE INDEX IF NOT EXISTS transactions_user_id_idx ON transactions (user_id);
CREATE INDEX IF NOT EXISTS transactions_source_id_idx ON transactions (source_id);
CREATE INDEX IF NOT EXISTS transactions_transaction_id_idx ON transactions (transaction_id);
CREATE INDEX IF NOT EXISTS sources_name_idx ON sources (name);

-- TRIGGERS --
CREATE OR REPLACE FUNCTION update_timestamp()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp
    BEFORE UPDATE
    ON transactions
    FOR EACH ROW
EXECUTE PROCEDURE update_timestamp();

-- DATA --
INSERT INTO sources (id, name)
VALUES ('8bd85576-8d8c-47b8-bfa8-6e9d2fc4d267', 'game'),
       ('cf086d5c-e99b-48bf-809b-e1b335a41886', 'server'),
       ('138075f8-059e-4fd9-a590-c85c8d97a33a', 'payment');
INSERT INTO users (id, balance, account_number)
VALUES ('f60ae2e1-ee72-4a6a-bef2-7cde5c83782f', 0, 'a68b6053-0fbc-4986-9364-584acd3ed9c0');

COMMIT;