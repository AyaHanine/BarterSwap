package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type createServiceInput struct {
	ProviderID   int
	Titre        string
	Description  string
	Categorie    string
	DureeMinutes int
	Credits      int
	Ville        string
	Actif        bool
}

type updateServiceInput struct {
	Titre        *string
	Description  *string
	Categorie    *string
	DureeMinutes *int
	Credits      *int
	Ville        *string
	Actif        *bool
}

type listServicesFilter struct {
	Categorie string
	Ville     string
	Search    string
}

func scanService(scanner interface {
	Scan(dest ...any) error
}) (Service, error) {
	var svc Service
	var createdAt time.Time
	err := scanner.Scan(
		&svc.ID, &svc.ProviderID, &svc.Titre, &svc.Description, &svc.Categorie,
		&svc.DureeMinutes, &svc.Credits, &svc.Ville, &svc.Actif, &createdAt,
	)
	if err != nil {
		return Service{}, err
	}
	svc.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return svc, nil
}

func (s *Store) CreateService(ctx context.Context, in createServiceInput) (Service, error) {
	const q = `
INSERT INTO services (provider_id, titre, description, categorie, duree_minutes, credits, ville, actif)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at`
	row := s.db.QueryRowContext(ctx, q,
		in.ProviderID, in.Titre, in.Description, in.Categorie,
		in.DureeMinutes, in.Credits, in.Ville, in.Actif,
	)
	svc, err := scanService(row)
	if err != nil {
		return Service{}, fmt.Errorf("insert service: %w", err)
	}
	return svc, nil
}

func (s *Store) GetService(ctx context.Context, id int) (Service, error) {
	const q = `
SELECT id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at
FROM services WHERE id = $1`
	svc, err := scanService(s.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return Service{}, ErrNotFound
	}
	if err != nil {
		return Service{}, fmt.Errorf("get service: %w", err)
	}
	return svc, nil
}

func (s *Store) UpdateService(ctx context.Context, id int, in updateServiceInput) (Service, error) {
	current, err := s.GetService(ctx, id)
	if err != nil {
		return Service{}, err
	}

	titre := current.Titre
	description := current.Description
	categorie := current.Categorie
	duree := current.DureeMinutes
	credits := current.Credits
	ville := current.Ville
	actif := current.Actif

	if in.Titre != nil {
		titre = *in.Titre
	}
	if in.Description != nil {
		description = *in.Description
	}
	if in.Categorie != nil {
		categorie = *in.Categorie
	}
	if in.DureeMinutes != nil {
		duree = *in.DureeMinutes
	}
	if in.Credits != nil {
		credits = *in.Credits
	}
	if in.Ville != nil {
		ville = *in.Ville
	}
	if in.Actif != nil {
		actif = *in.Actif
	}

	const q = `
UPDATE services
SET titre = $1, description = $2, categorie = $3, duree_minutes = $4, credits = $5, ville = $6, actif = $7
WHERE id = $8
RETURNING id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at`
	svc, err := scanService(s.db.QueryRowContext(ctx, q, titre, description, categorie, duree, credits, ville, actif, id))
	if err != nil {
		return Service{}, fmt.Errorf("update service: %w", err)
	}
	return svc, nil
}

func (s *Store) DeleteService(ctx context.Context, id int) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM services WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListServices(ctx context.Context, f listServicesFilter) ([]Service, error) {
	var (
		b    strings.Builder
		args []any
	)
	b.WriteString(`
SELECT id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at
FROM services WHERE 1=1`)

	if f.Categorie != "" {
		args = append(args, f.Categorie)
		fmt.Fprintf(&b, ` AND categorie = $%d`, len(args))
	}
	if f.Ville != "" {
		args = append(args, f.Ville)
		fmt.Fprintf(&b, ` AND ville ILIKE $%d`, len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		fmt.Fprintf(&b, ` AND (titre ILIKE $%d OR description ILIKE $%d)`, len(args), len(args))
	}
	b.WriteString(` ORDER BY created_at DESC, id DESC`)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	services := make([]Service, 0)
	for rows.Next() {
		svc, err := scanService(rows)
		if err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		services = append(services, svc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *Store) UserHasSkill(ctx context.Context, userID int, skillName string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM skills WHERE user_id = $1 AND lower(nom) = lower($2)`,
		userID, skillName,
	).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
