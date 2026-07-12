package main

import (
	"context"
	"fmt"
	"strings"
)

type createExchangeRequestData struct {
	ServiceID int
}

// CreateExchange crée une demande d'échange pour un service.
func (a *App) CreateExchange(ctx context.Context, actorID int, in createExchangeRequestData) (Exchange, error) {
	if actorID <= 0 {
		return Exchange{}, ErrUnauthorized
	}
	if in.ServiceID <= 0 {
		return Exchange{}, fmt.Errorf("%w: service_id invalide", ErrValidation)
	}

	svc, err := a.store.GetService(ctx, in.ServiceID)
	if err != nil {
		return Exchange{}, err
	}
	if !svc.Actif {
		return Exchange{}, fmt.Errorf("%w: le service n'est pas actif", ErrValidation)
	}
	if svc.ProviderID == actorID {
		return Exchange{}, fmt.Errorf("%w: impossible de demander son propre service", ErrValidation)
	}

	if _, err := a.store.GetUser(ctx, actorID); err != nil {
		return Exchange{}, err
	}

	balance, err := a.store.GetUserCreditBalance(ctx, actorID)
	if err != nil {
		return Exchange{}, err
	}
	if balance < svc.Credits {
		return Exchange{}, fmt.Errorf("%w: crédits insuffisants", ErrValidation)
	}

	return a.store.CreateExchange(ctx, createExchangeInput{
		ServiceID:   svc.ID,
		RequesterID: actorID,
		OwnerID:     svc.ProviderID,
		Credits:     svc.Credits,
	})
}

// GetExchange retourne le détail d'un échange.
func (a *App) GetExchange(ctx context.Context, id int) (Exchange, error) {
	if id <= 0 {
		return Exchange{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	return a.store.GetExchange(ctx, id)
}

// ListExchanges liste les échanges de l'utilisateur authentifié (envoyés + reçus).
func (a *App) ListExchanges(ctx context.Context, actorID int, status string) ([]Exchange, error) {
	if actorID <= 0 {
		return nil, ErrUnauthorized
	}
	status = strings.TrimSpace(status)
	if status != "" {
		if !isValidExchangeStatus(status) {
			return nil, fmt.Errorf("%w: statut invalide", ErrValidation)
		}
	}
	return a.store.ListExchanges(ctx, listExchangesFilter{
		UserID: actorID,
		Status: status,
	})
}

// AcceptExchange accepte une demande (offreur uniquement) et bloque les crédits du demandeur.
func (a *App) AcceptExchange(ctx context.Context, actorID, exchangeID int) (Exchange, error) {
	if actorID <= 0 {
		return Exchange{}, ErrUnauthorized
	}
	if exchangeID <= 0 {
		return Exchange{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	ex, err := a.store.GetExchange(ctx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if ex.OwnerID != actorID {
		return Exchange{}, ErrForbidden
	}
	return a.store.AcceptExchange(ctx, exchangeID)
}

// RejectExchange refuse une demande (offreur uniquement).
func (a *App) RejectExchange(ctx context.Context, actorID, exchangeID int) (Exchange, error) {
	if actorID <= 0 {
		return Exchange{}, ErrUnauthorized
	}
	if exchangeID <= 0 {
		return Exchange{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	ex, err := a.store.GetExchange(ctx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if ex.OwnerID != actorID {
		return Exchange{}, ErrForbidden
	}
	return a.store.RejectExchange(ctx, exchangeID)
}

// CompleteExchange marque un échange accepté comme terminé et crédite l'offreur.
func (a *App) CompleteExchange(ctx context.Context, actorID, exchangeID int) (Exchange, error) {
	if actorID <= 0 {
		return Exchange{}, ErrUnauthorized
	}
	if exchangeID <= 0 {
		return Exchange{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	ex, err := a.store.GetExchange(ctx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if ex.RequesterID != actorID && ex.OwnerID != actorID {
		return Exchange{}, ErrForbidden
	}
	return a.store.CompleteExchange(ctx, exchangeID)
}

// CancelExchange annule une demande (demandeur ou offreur) et restitue les crédits si bloqués.
func (a *App) CancelExchange(ctx context.Context, actorID, exchangeID int) (Exchange, error) {
	if actorID <= 0 {
		return Exchange{}, ErrUnauthorized
	}
	if exchangeID <= 0 {
		return Exchange{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	ex, err := a.store.GetExchange(ctx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if ex.RequesterID != actorID && ex.OwnerID != actorID {
		return Exchange{}, ErrForbidden
	}
	return a.store.CancelExchange(ctx, exchangeID)
}

func isValidExchangeStatus(status string) bool {
	switch status {
	case exchangeStatusPending, exchangeStatusAccepted, exchangeStatusRejected,
		exchangeStatusCompleted, exchangeStatusCancelled:
		return true
	default:
		return false
	}
}
