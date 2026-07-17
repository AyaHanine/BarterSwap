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

const welcomeCredits = 10

type createUserInput struct {
	Pseudo string
	Bio    string
	Ville  string
}

type updateUserInput struct {
	Pseudo *string
	Bio    *string
	Ville  *string
}

func (s *Store) CreateUser(ctx context.Context, in createUserInput) (User, error) {
	var user User
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		const q = `
INSERT INTO users (pseudo, bio, ville, credit_balance)
VALUES ($1, $2, $3, $4)
RETURNING id, pseudo, bio, ville, credit_balance, created_at`
		var createdAt time.Time
		err := tx.QueryRowContext(ctx, q, in.Pseudo, in.Bio, in.Ville, welcomeCredits).Scan(
			&user.ID, &user.Pseudo, &user.Bio, &user.Ville, &user.CreditBalance, &createdAt,
		)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				return fmt.Errorf("%w: pseudo déjà utilisé", ErrConflict)
			}
			return fmt.Errorf("insert user: %w", err)
		}
		user.CreatedAt = createdAt.UTC().Format(time.RFC3339)

		const tq = `
INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
VALUES ($1, NULL, $2, 'earn')`
		if _, err := tx.ExecContext(ctx, tq, user.ID, welcomeCredits); err != nil {
			return fmt.Errorf("insert welcome credit: %w", err)
		}
		return nil
	})
	if err != nil {
		return User{}, err
	}
	user.Skills = []Skill{}
	return user, nil
}

func (s *Store) GetUser(ctx context.Context, id int) (User, error) {
	const q = `
SELECT id, pseudo, bio, ville, credit_balance, created_at
FROM users WHERE id = $1`
	var user User
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&user.ID, &user.Pseudo, &user.Bio, &user.Ville, &user.CreditBalance, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	user.CreatedAt = createdAt.UTC().Format(time.RFC3339)

	skills, err := s.ListSkills(ctx, id)
	if err != nil {
		return User{}, err
	}
	user.Skills = skills
	return user, nil
}

func (s *Store) UpdateUser(ctx context.Context, id int, in updateUserInput) (User, error) {
	current, err := s.GetUser(ctx, id)
	if err != nil {
		return User{}, err
	}

	pseudo := current.Pseudo
	bio := current.Bio
	ville := current.Ville
	if in.Pseudo != nil {
		pseudo = *in.Pseudo
	}
	if in.Bio != nil {
		bio = *in.Bio
	}
	if in.Ville != nil {
		ville = *in.Ville
	}

	const q = `
UPDATE users SET pseudo = $1, bio = $2, ville = $3
WHERE id = $4
RETURNING id, pseudo, bio, ville, credit_balance, created_at`
	var user User
	var createdAt time.Time
	err = s.db.QueryRowContext(ctx, q, pseudo, bio, ville, id).Scan(
		&user.ID, &user.Pseudo, &user.Bio, &user.Ville, &user.CreditBalance, &createdAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return User{}, fmt.Errorf("%w: pseudo déjà utilisé", ErrConflict)
		}
		return User{}, fmt.Errorf("update user: %w", err)
	}
	user.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	user.Skills = current.Skills
	return user, nil
}

func (s *Store) ListSkills(ctx context.Context, userID int) ([]Skill, error) {
	const q = `SELECT nom, niveau FROM skills WHERE user_id = $1 ORDER BY nom`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	skills := make([]Skill, 0)
	for rows.Next() {
		var sk Skill
		if err := rows.Scan(&sk.Nom, &sk.Niveau); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		skills = append(skills, sk)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return skills, nil
}

func (s *Store) ReplaceSkills(ctx context.Context, userID int, skills []Skill) ([]Skill, error) {
	exists, err := s.userExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	err = s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM skills WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("delete skills: %w", err)
		}
		const iq = `INSERT INTO skills (user_id, nom, niveau) VALUES ($1, $2, $3)`
		for _, sk := range skills {
			if _, err := tx.ExecContext(ctx, iq, userID, sk.Nom, sk.Niveau); err != nil {
				return fmt.Errorf("insert skill: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.ListSkills(ctx, userID)
}

func (s *Store) userExists(ctx context.Context, id int) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = $1`, id).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetUserStats calcule les statistiques du tableau de bord d'un utilisateur.
func (s *Store) GetUserStats(ctx context.Context, userID int) (UserStats, error) {
	exists, err := s.userExists(ctx, userID)
	if err != nil {
		return UserStats{}, err
	}
	if !exists {
		return UserStats{}, ErrNotFound
	}

	stats := UserStats{UserID: userID}

	err = s.db.QueryRowContext(ctx,
		`SELECT credit_balance FROM users WHERE id = $1`, userID,
	).Scan(&stats.CreditBalance)
	if err != nil {
		return UserStats{}, fmt.Errorf("get credit balance: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM services WHERE provider_id = $1 AND actif = TRUE`, userID,
	).Scan(&stats.ServicesActifs)
	if err != nil {
		return UserStats{}, fmt.Errorf("count active services: %w", err)
	}

	err = s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM exchanges
WHERE status = 'completed' AND (requester_id = $1 OR owner_id = $1)`, userID,
	).Scan(&stats.EchangesCompletes)
	if err != nil {
		return UserStats{}, fmt.Errorf("count completed exchanges: %w", err)
	}

	var avg sql.NullFloat64
	err = s.db.QueryRowContext(ctx, `
SELECT AVG(note)::float8, COUNT(*) FROM reviews WHERE target_id = $1`, userID,
	).Scan(&avg, &stats.NbAvis)
	if err != nil {
		return UserStats{}, fmt.Errorf("review stats: %w", err)
	}
	if avg.Valid {
		stats.NoteMoyenne = avg.Float64
	}

	// total_gagne : crédits gagnés via des échanges (hors crédits de bienvenue).
	// total_depense : dépenses nettes = spend − refund.
	err = s.db.QueryRowContext(ctx, `
SELECT
  COALESCE(SUM(CASE WHEN type = 'earn' AND exchange_id IS NOT NULL THEN montant ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN type = 'spend' THEN ABS(montant) ELSE 0 END), 0)
    - COALESCE(SUM(CASE WHEN type = 'refund' THEN montant ELSE 0 END), 0)
FROM credit_transactions
WHERE user_id = $1`, userID,
	).Scan(&stats.TotalGagne, &stats.TotalDepense)
	if err != nil {
		return UserStats{}, fmt.Errorf("credit totals: %w", err)
	}

	return stats, nil
}

func normalizePseudo(pseudo string) string {
	return strings.TrimSpace(pseudo)
}
