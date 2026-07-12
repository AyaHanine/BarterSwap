package main

import (
	"context"
	"net/http"
)

type createExchangeHTTPRequest struct {
	ServiceID int `json:"service_id"`
}

func (app *App) handleCreateExchange(w http.ResponseWriter, r *http.Request) {
	actorID, err := userIDFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "header X-User-ID requis")
		return
	}

	var req createExchangeHTTPRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	ex, err := app.CreateExchange(r.Context(), actorID, createExchangeRequestData{
		ServiceID: req.ServiceID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ex)
}

func (app *App) handleListExchanges(w http.ResponseWriter, r *http.Request) {
	actorID, err := userIDFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "header X-User-ID requis")
		return
	}

	exchanges, err := app.ListExchanges(r.Context(), actorID, r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exchanges)
}

func (app *App) handleGetExchange(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}

	ex, err := app.GetExchange(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ex)
}

func (app *App) handleAcceptExchange(w http.ResponseWriter, r *http.Request) {
	app.handleExchangeAction(w, r, app.AcceptExchange)
}

func (app *App) handleRejectExchange(w http.ResponseWriter, r *http.Request) {
	app.handleExchangeAction(w, r, app.RejectExchange)
}

func (app *App) handleCompleteExchange(w http.ResponseWriter, r *http.Request) {
	app.handleExchangeAction(w, r, app.CompleteExchange)
}

func (app *App) handleCancelExchange(w http.ResponseWriter, r *http.Request) {
	app.handleExchangeAction(w, r, app.CancelExchange)
}

func (app *App) handleExchangeAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(context.Context, int, int) (Exchange, error),
) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	actorID, err := userIDFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "header X-User-ID requis")
		return
	}

	ex, err := action(r.Context(), actorID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ex)
}
