package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	exchangeStatusPending   = "pending"
	exchangeStatusAccepted  = "accepted"
	exchangeStatusRejected  = "rejected"
	exchangeStatusCompleted = "completed"
	exchangeStatusCancelled = "cancelled"
)

type createExchangeInput struct {
	ServiceID   int
	RequesterID int
	OwnerID     int
	Credits     int
}

type listExchangesFilter struct {
	UserID int
	Status string
}

func scanExchange(scanner interface {
	Scan(dest ...any) error
}) (Exchange, error) {
	var ex Exchange
	var credits int
	var createdAt, updatedAt time.Time
	err := scanner.Scan(
		&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID,
		&credits, &ex.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return Exchange{}, err
	}
	ex.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	ex.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return ex, nil
}

func scanExchangeWithCredits(scanner interface {
	Scan(dest ...any) error
}) (Exchange, int, error) {
	var ex Exchange
	var credits int
	var createdAt, updatedAt time.Time
	err := scanner.Scan(
		&ex.ID, &ex.ServiceID, &ex.RequesterID, &ex.OwnerID,
		&credits, &ex.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return Exchange{}, 0, err
	}
	ex.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	ex.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return ex, credits, nil
}

func (s *Store) CreateExchange(ctx context.Context, in createExchangeInput) (Exchange, error) {
	const q = `
INSERT INTO exchanges (service_id, requester_id, owner_id, credits)
VALUES ($1, $2, $3, $4)
RETURNING id, service_id, requester_id, owner_id, credits, status, created_at, updated_at`
	ex, err := scanExchange(s.db.QueryRowContext(ctx, q,
		in.ServiceID, in.RequesterID, in.OwnerID, in.Credits,
	))
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return Exchange{}, fmt.Errorf("%w: le service est déjà réservé", ErrConflict)
		}
		return Exchange{}, fmt.Errorf("insert exchange: %w", err)
	}
	return ex, nil
}

func (s *Store) GetExchange(ctx context.Context, id int) (Exchange, error) {
	const q = `
SELECT id, service_id, requester_id, owner_id, credits, status, created_at, updated_at
FROM exchanges WHERE id = $1`
	ex, err := scanExchange(s.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return Exchange{}, ErrNotFound
	}
	if err != nil {
		return Exchange{}, fmt.Errorf("get exchange: %w", err)
	}
	return ex, nil
}

func (s *Store) ListExchanges(ctx context.Context, f listExchangesFilter) ([]Exchange, error) {
	var (
		b    strings.Builder
		args []any
	)
	b.WriteString(`
SELECT id, service_id, requester_id, owner_id, credits, status, created_at, updated_at
FROM exchanges WHERE 1=1`)

	if f.UserID > 0 {
		args = append(args, f.UserID)
		fmt.Fprintf(&b, ` AND (requester_id = $%d OR owner_id = $%d)`, len(args), len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		fmt.Fprintf(&b, ` AND status = $%d`, len(args))
	}
	b.WriteString(` ORDER BY created_at DESC, id DESC`)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list exchanges: %w", err)
	}
	defer rows.Close()

	exchanges := make([]Exchange, 0)
	for rows.Next() {
		ex, err := scanExchange(rows)
		if err != nil {
			return nil, fmt.Errorf("scan exchange: %w", err)
		}
		exchanges = append(exchanges, ex)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return exchanges, nil
}

func (s *Store) GetUserCreditBalance(ctx context.Context, userID int) (int, error) {
	var balance int
	err := s.db.QueryRowContext(ctx, `SELECT credit_balance FROM users WHERE id = $1`, userID).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get credit balance: %w", err)
	}
	return balance, nil
}

func (s *Store) AcceptExchange(ctx context.Context, exchangeID int) (Exchange, error) {
	return s.transitionExchange(ctx, exchangeID, func(ctx context.Context, tx *sql.Tx, ex Exchange, credits int) error {
		if ex.Status != exchangeStatusPending {
			return fmt.Errorf("%w: l'échange n'est pas en attente", ErrConflict)
		}
		if err := blockCreditsTx(ctx, tx, ex.RequesterID, exchangeID, credits); err != nil {
			return err
		}
		return updateExchangeStatusTx(ctx, tx, exchangeID, exchangeStatusAccepted)
	})
}

func (s *Store) RejectExchange(ctx context.Context, exchangeID int) (Exchange, error) {
	return s.transitionExchange(ctx, exchangeID, func(ctx context.Context, tx *sql.Tx, ex Exchange, credits int) error {
		if ex.Status != exchangeStatusPending {
			return fmt.Errorf("%w: l'échange n'est pas en attente", ErrConflict)
		}
		return updateExchangeStatusTx(ctx, tx, exchangeID, exchangeStatusRejected)
	})
}

func (s *Store) CancelExchange(ctx context.Context, exchangeID int) (Exchange, error) {
	return s.transitionExchange(ctx, exchangeID, func(ctx context.Context, tx *sql.Tx, ex Exchange, credits int) error {
		if ex.Status != exchangeStatusPending && ex.Status != exchangeStatusAccepted {
			return fmt.Errorf("%w: l'échange ne peut plus être annulé", ErrConflict)
		}
		if ex.Status == exchangeStatusAccepted {
			if err := refundCreditsTx(ctx, tx, ex.RequesterID, exchangeID, credits); err != nil {
				return err
			}
		}
		return updateExchangeStatusTx(ctx, tx, exchangeID, exchangeStatusCancelled)
	})
}

func (s *Store) CompleteExchange(ctx context.Context, exchangeID int) (Exchange, error) {
	return s.transitionExchange(ctx, exchangeID, func(ctx context.Context, tx *sql.Tx, ex Exchange, credits int) error {
		if ex.Status != exchangeStatusAccepted {
			return fmt.Errorf("%w: l'échange doit être accepté avant d'être terminé", ErrConflict)
		}
		if err := creditOwnerTx(ctx, tx, ex.OwnerID, exchangeID, credits); err != nil {
			return err
		}
		return updateExchangeStatusTx(ctx, tx, exchangeID, exchangeStatusCompleted)
	})
}

func (s *Store) transitionExchange(ctx context.Context, exchangeID int, fn func(context.Context, *sql.Tx, Exchange, int) error) (Exchange, error) {
	var result Exchange
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		ex, credits, err := getExchangeForUpdate(ctx, tx, exchangeID)
		if err != nil {
			return err
		}
		if err := fn(ctx, tx, ex, credits); err != nil {
			return err
		}
		result, err = getExchangeTx(ctx, tx, exchangeID)
		return err
	})
	if err != nil {
		return Exchange{}, err
	}
	return result, nil
}

func getExchangeForUpdate(ctx context.Context, tx *sql.Tx, id int) (Exchange, int, error) {
	const q = `
SELECT id, service_id, requester_id, owner_id, credits, status, created_at, updated_at
FROM exchanges WHERE id = $1 FOR UPDATE`
	ex, credits, err := scanExchangeWithCredits(tx.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return Exchange{}, 0, ErrNotFound
	}
	if err != nil {
		return Exchange{}, 0, fmt.Errorf("get exchange for update: %w", err)
	}
	return ex, credits, nil
}

func getExchangeTx(ctx context.Context, tx *sql.Tx, id int) (Exchange, error) {
	const q = `
SELECT id, service_id, requester_id, owner_id, credits, status, created_at, updated_at
FROM exchanges WHERE id = $1`
	ex, err := scanExchange(tx.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return Exchange{}, ErrNotFound
	}
	return ex, err
}

func blockCreditsTx(ctx context.Context, tx *sql.Tx, requesterID, exchangeID, credits int) error {
	var balance int
	err := tx.QueryRowContext(ctx,
		`SELECT credit_balance FROM users WHERE id = $1 FOR UPDATE`, requesterID,
	).Scan(&balance)
	if err != nil {
		return fmt.Errorf("lock requester: %w", err)
	}
	if balance < credits {
		return fmt.Errorf("%w: crédits insuffisants pour le demandeur", ErrValidation)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET credit_balance = credit_balance - $1 WHERE id = $2`,
		credits, requesterID,
	); err != nil {
		return fmt.Errorf("block credits: %w", err)
	}
	const q = `
INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
VALUES ($1, $2, $3, 'spend')`
	if _, err := tx.ExecContext(ctx, q, requesterID, exchangeID, -credits); err != nil {
		return fmt.Errorf("insert spend transaction: %w", err)
	}
	return nil
}

func refundCreditsTx(ctx context.Context, tx *sql.Tx, requesterID, exchangeID, credits int) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET credit_balance = credit_balance + $1 WHERE id = $2`,
		credits, requesterID,
	); err != nil {
		return fmt.Errorf("refund credits: %w", err)
	}
	const q = `
INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
VALUES ($1, $2, $3, 'refund')`
	if _, err := tx.ExecContext(ctx, q, requesterID, exchangeID, credits); err != nil {
		return fmt.Errorf("insert refund transaction: %w", err)
	}
	return nil
}

func creditOwnerTx(ctx context.Context, tx *sql.Tx, ownerID, exchangeID, credits int) error {
	if _, err := tx.ExecContext(ctx,
		`SELECT id FROM users WHERE id = $1 FOR UPDATE`, ownerID,
	); err != nil {
		return fmt.Errorf("lock owner: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET credit_balance = credit_balance + $1 WHERE id = $2`,
		credits, ownerID,
	); err != nil {
		return fmt.Errorf("credit owner: %w", err)
	}
	const q = `
INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
VALUES ($1, $2, $3, 'earn')`
	if _, err := tx.ExecContext(ctx, q, ownerID, exchangeID, credits); err != nil {
		return fmt.Errorf("insert earn transaction: %w", err)
	}
	return nil
}

func updateExchangeStatusTx(ctx context.Context, tx *sql.Tx, id int, status string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE exchanges SET status = $1, updated_at = NOW() WHERE id = $2`, status, id,
	)
	if err != nil {
		return fmt.Errorf("update exchange status: %w", err)
	}
	return nil
}
