package main

import (
	"context"
	"database/sql"
)

// Store encapsule l'accès à la base de données via database/sql.
type Store struct {
	db *sql.DB
}

// NewStore crée un Store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
