package main

import (
	"net/http"
)

type createReviewHTTPRequest struct {
	Note        int    `json:"note"`
	Commentaire string `json:"commentaire"`
}

func (app *App) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	exchangeID, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	actorID, err := userIDFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "header X-User-ID requis")
		return
	}

	var req createReviewHTTPRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	rv, err := app.CreateReview(r.Context(), actorID, exchangeID, createReviewRequestData{
		Note:        req.Note,
		Commentaire: req.Commentaire,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rv)
}

func (app *App) handleListUserReviews(w http.ResponseWriter, r *http.Request) {
	userID, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}

	reviews, err := app.ListUserReviews(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (app *App) handleListServiceReviews(w http.ResponseWriter, r *http.Request) {
	serviceID, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}

	reviews, err := app.ListServiceReviews(r.Context(), serviceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}
