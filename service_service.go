package main

import (
	"context"
	"fmt"
	"strings"
)

// Catégories d'échange (liste fermée de l'énoncé).
var allowedCategories = map[string]struct{}{
	"Informatique": {},
	"Jardinage":    {},
	"Bricolage":    {},
	"Cuisine":      {},
	"Musique":      {},
	"Langues":      {},
	"Sport":        {},
	"Tutorat":      {},
	"Déménagement": {},
	"Photographie": {},
	"Animalier":    {},
	"Couture":      {},
	"Autre":        {},
}

type createServiceRequestData struct {
	Titre        string
	Description  string
	Categorie    string
	DureeMinutes int
	Credits      int
	Ville        string
	Actif        *bool
}

type updateServiceRequestData struct {
	Titre        *string
	Description  *string
	Categorie    *string
	DureeMinutes *int
	Credits      *int
	Ville        *string
	Actif        *bool
}

// CreateService crée une annonce liée à une compétence de l'utilisateur.
func (a *App) CreateService(ctx context.Context, actorID int, in createServiceRequestData) (Service, error) {
	if actorID <= 0 {
		return Service{}, ErrUnauthorized
	}

	titre := strings.TrimSpace(in.Titre)
	if titre == "" {
		return Service{}, fmt.Errorf("%w: titre obligatoire", ErrValidation)
	}
	categorie := strings.TrimSpace(in.Categorie)
	if _, ok := allowedCategories[categorie]; !ok {
		return Service{}, fmt.Errorf("%w: catégorie invalide", ErrValidation)
	}
	if in.DureeMinutes <= 0 {
		return Service{}, fmt.Errorf("%w: duree_minutes doit être > 0", ErrValidation)
	}
	if in.Credits <= 0 {
		return Service{}, fmt.Errorf("%w: credits doit être > 0", ErrValidation)
	}

	if _, err := a.store.GetUser(ctx, actorID); err != nil {
		return Service{}, err
	}

	hasSkill, err := a.store.UserHasSkill(ctx, actorID, categorie)
	if err != nil {
		return Service{}, err
	}
	if !hasSkill {
		return Service{}, fmt.Errorf("%w: l'utilisateur n'a pas la compétence %q", ErrValidation, categorie)
	}

	actif := true
	if in.Actif != nil {
		actif = *in.Actif
	}

	return a.store.CreateService(ctx, createServiceInput{
		ProviderID:   actorID,
		Titre:        titre,
		Description:  strings.TrimSpace(in.Description),
		Categorie:    categorie,
		DureeMinutes: in.DureeMinutes,
		Credits:      in.Credits,
		Ville:        strings.TrimSpace(in.Ville),
		Actif:        actif,
	})
}

// GetService retourne le détail d'une annonce.
func (a *App) GetService(ctx context.Context, id int) (Service, error) {
	if id <= 0 {
		return Service{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	return a.store.GetService(ctx, id)
}

// ListServices liste les annonces avec filtres côté serveur.
func (a *App) ListServices(ctx context.Context, categorie, ville, search string) ([]Service, error) {
	return a.store.ListServices(ctx, listServicesFilter{
		Categorie: strings.TrimSpace(categorie),
		Ville:     strings.TrimSpace(ville),
		Search:    strings.TrimSpace(search),
	})
}

// UpdateService modifie une annonce appartenant à l'utilisateur authentifié.
func (a *App) UpdateService(ctx context.Context, actorID, serviceID int, in updateServiceRequestData) (Service, error) {
	if serviceID <= 0 {
		return Service{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if actorID <= 0 {
		return Service{}, ErrUnauthorized
	}

	current, err := a.store.GetService(ctx, serviceID)
	if err != nil {
		return Service{}, err
	}
	if current.ProviderID != actorID {
		return Service{}, ErrForbidden
	}

	upd := updateServiceInput{}

	if in.Titre != nil {
		t := strings.TrimSpace(*in.Titre)
		if t == "" {
			return Service{}, fmt.Errorf("%w: titre obligatoire", ErrValidation)
		}
		upd.Titre = &t
	}
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		upd.Description = &d
	}
	if in.Categorie != nil {
		c := strings.TrimSpace(*in.Categorie)
		if _, ok := allowedCategories[c]; !ok {
			return Service{}, fmt.Errorf("%w: catégorie invalide", ErrValidation)
		}
		hasSkill, err := a.store.UserHasSkill(ctx, actorID, c)
		if err != nil {
			return Service{}, err
		}
		if !hasSkill {
			return Service{}, fmt.Errorf("%w: l'utilisateur n'a pas la compétence %q", ErrValidation, c)
		}
		upd.Categorie = &c
	}
	if in.DureeMinutes != nil {
		if *in.DureeMinutes <= 0 {
			return Service{}, fmt.Errorf("%w: duree_minutes doit être > 0", ErrValidation)
		}
		upd.DureeMinutes = in.DureeMinutes
	}
	if in.Credits != nil {
		if *in.Credits <= 0 {
			return Service{}, fmt.Errorf("%w: credits doit être > 0", ErrValidation)
		}
		upd.Credits = in.Credits
	}
	if in.Ville != nil {
		v := strings.TrimSpace(*in.Ville)
		upd.Ville = &v
	}
	if in.Actif != nil {
		upd.Actif = in.Actif
	}

	return a.store.UpdateService(ctx, serviceID, upd)
}

// DeleteService supprime une annonce appartenant à l'utilisateur authentifié.
func (a *App) DeleteService(ctx context.Context, actorID, serviceID int) error {
	if serviceID <= 0 {
		return fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if actorID <= 0 {
		return ErrUnauthorized
	}
	current, err := a.store.GetService(ctx, serviceID)
	if err != nil {
		return err
	}
	if current.ProviderID != actorID {
		return ErrForbidden
	}
	return a.store.DeleteService(ctx, serviceID)
}
