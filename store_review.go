package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type createReviewInput struct {
	ExchangeID  int
	AuthorID    int
	TargetID    int
	ServiceID   int
	Note        int
	Commentaire string
}

func scanReview(scanner interface {
	Scan(dest ...any) error
}) (Review, error) {
	var rv Review
	var createdAt time.Time
	err := scanner.Scan(
		&rv.ID, &rv.ExchangeID, &rv.AuthorID, &rv.TargetID, &rv.ServiceID,
		&rv.Note, &rv.Commentaire, &createdAt,
	)
	if err != nil {
		return Review{}, err
	}
	rv.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return rv, nil
}

func (s *Store) CreateReview(ctx context.Context, in createReviewInput) (Review, error) {
	const q = `
INSERT INTO reviews (exchange_id, author_id, target_id, service_id, note, commentaire)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, exchange_id, author_id, target_id, service_id, note, commentaire, created_at`
	rv, err := scanReview(s.db.QueryRowContext(ctx, q,
		in.ExchangeID, in.AuthorID, in.TargetID, in.ServiceID, in.Note, in.Commentaire,
	))
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return Review{}, fmt.Errorf("%w: avis déjà laissé pour cet échange", ErrConflict)
		}
		return Review{}, fmt.Errorf("insert review: %w", err)
	}
	return rv, nil
}

func (s *Store) ListReviewsByTarget(ctx context.Context, targetID int) ([]Review, error) {
	const q = `
SELECT id, exchange_id, author_id, target_id, service_id, note, commentaire, created_at
FROM reviews WHERE target_id = $1
ORDER BY created_at DESC, id DESC`
	return s.queryReviews(ctx, q, targetID)
}

func (s *Store) ListReviewsByService(ctx context.Context, serviceID int) ([]Review, error) {
	const q = `
SELECT id, exchange_id, author_id, target_id, service_id, note, commentaire, created_at
FROM reviews WHERE service_id = $1
ORDER BY created_at DESC, id DESC`
	return s.queryReviews(ctx, q, serviceID)
}

func (s *Store) queryReviews(ctx context.Context, q string, arg int) ([]Review, error) {
	rows, err := s.db.QueryContext(ctx, q, arg)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]Review, 0)
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reviews, nil
}

func (s *Store) UserExists(ctx context.Context, id int) error {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = $1`, id).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("user exists: %w", err)
	}
	return nil
}

func (s *Store) ServiceExists(ctx context.Context, id int) error {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM services WHERE id = $1`, id).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("service exists: %w", err)
	}
	return nil
}
