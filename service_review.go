package main

import (
	"context"
	"fmt"
	"strings"
)

type createReviewRequestData struct {
	Note        int
	Commentaire string
}

// CreateReview enregistre un avis sur un échange terminé.
func (a *App) CreateReview(ctx context.Context, actorID, exchangeID int, in createReviewRequestData) (Review, error) {
	if actorID <= 0 {
		return Review{}, ErrUnauthorized
	}
	if exchangeID <= 0 {
		return Review{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if in.Note < 1 || in.Note > 5 {
		return Review{}, fmt.Errorf("%w: la note doit être entre 1 et 5", ErrValidation)
	}

	ex, err := a.store.GetExchange(ctx, exchangeID)
	if err != nil {
		return Review{}, err
	}
	if ex.Status != exchangeStatusCompleted {
		return Review{}, fmt.Errorf("%w: l'échange doit être terminé pour laisser un avis", ErrValidation)
	}
	if ex.RequesterID != actorID && ex.OwnerID != actorID {
		return Review{}, ErrForbidden
	}

	targetID := ex.OwnerID
	if actorID == ex.OwnerID {
		targetID = ex.RequesterID
	}

	return a.store.CreateReview(ctx, createReviewInput{
		ExchangeID:  exchangeID,
		AuthorID:    actorID,
		TargetID:    targetID,
		ServiceID:   ex.ServiceID,
		Note:        in.Note,
		Commentaire: strings.TrimSpace(in.Commentaire),
	})
}

// ListUserReviews retourne les avis reçus par un utilisateur.
func (a *App) ListUserReviews(ctx context.Context, userID int) ([]Review, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if err := a.store.UserExists(ctx, userID); err != nil {
		return nil, err
	}
	return a.store.ListReviewsByTarget(ctx, userID)
}

// ListServiceReviews retourne les avis liés à un service.
func (a *App) ListServiceReviews(ctx context.Context, serviceID int) ([]Review, error) {
	if serviceID <= 0 {
		return nil, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if err := a.store.ServiceExists(ctx, serviceID); err != nil {
		return nil, err
	}
	return a.store.ListReviewsByService(ctx, serviceID)
}
