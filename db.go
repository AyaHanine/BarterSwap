package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
    id              SERIAL PRIMARY KEY,
    pseudo          TEXT NOT NULL UNIQUE,
    bio             TEXT NOT NULL DEFAULT '',
    ville           TEXT NOT NULL DEFAULT '',
    credit_balance  INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS skills (
    id       SERIAL PRIMARY KEY,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nom      TEXT NOT NULL,
    niveau   TEXT NOT NULL,
    UNIQUE (user_id, nom)
);

CREATE TABLE IF NOT EXISTS credit_transactions (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange_id  INTEGER,
    montant      INTEGER NOT NULL,
    type         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS services (
    id              SERIAL PRIMARY KEY,
    provider_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    titre           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    categorie       TEXT NOT NULL,
    duree_minutes   INTEGER NOT NULL,
    credits         INTEGER NOT NULL,
    ville           TEXT NOT NULL DEFAULT '',
    actif           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_services_categorie ON services (categorie);
CREATE INDEX IF NOT EXISTS idx_services_ville ON services (ville);
CREATE INDEX IF NOT EXISTS idx_services_provider ON services (provider_id);

CREATE TABLE IF NOT EXISTS exchanges (
    id            SERIAL PRIMARY KEY,
    service_id    INTEGER NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    requester_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credits       INTEGER NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exchanges_requester ON exchanges (requester_id);
CREATE INDEX IF NOT EXISTS idx_exchanges_owner ON exchanges (owner_id);
CREATE INDEX IF NOT EXISTS idx_exchanges_service ON exchanges (service_id);
CREATE INDEX IF NOT EXISTS idx_exchanges_status ON exchanges (status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_exchanges_one_active_per_service
ON exchanges (service_id) WHERE status IN ('pending', 'accepted');
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return migrateExchanges(db)
}

func migrateExchanges(db *sql.DB) error {
	const legacy = `
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'exchanges' AND column_name = 'provider_id'
    ) THEN
        ALTER TABLE exchanges RENAME COLUMN provider_id TO owner_id;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'exchanges' AND column_name = 'message'
    ) THEN
        ALTER TABLE exchanges DROP COLUMN message;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_exchanges_one_active_per_service
ON exchanges (service_id) WHERE status IN ('pending', 'accepted');
`
	if _, err := db.Exec(legacy); err != nil {
		return fmt.Errorf("migrate exchanges: %w", err)
	}
	return nil
}
