package main

import (
	"net/http"
)

type createServiceHTTPRequest struct {
	Titre        string `json:"titre"`
	Description  string `json:"description"`
	Categorie    string `json:"categorie"`
	DureeMinutes int    `json:"duree_minutes"`
	Credits      int    `json:"credits"`
	Ville        string `json:"ville"`
	Actif        *bool  `json:"actif"`
}

type updateServiceHTTPRequest struct {
	Titre        *string `json:"titre"`
	Description  *string `json:"description"`
	Categorie    *string `json:"categorie"`
	DureeMinutes *int    `json:"duree_minutes"`
	Credits      *int    `json:"credits"`
	Ville        *string `json:"ville"`
	Actif        *bool   `json:"actif"`
}

func (app *App) handleCreateService(w http.ResponseWriter, r *http.Request) {
	actorID, err := userIDFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "header X-User-ID requis")
		return
	}

	var req createServiceHTTPRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	svc, err := app.CreateService(r.Context(), actorID, createServiceRequestData{
		Titre:        req.Titre,
		Description:  req.Description,
		Categorie:    req.Categorie,
		DureeMinutes: req.DureeMinutes,
		Credits:      req.Credits,
		Ville:        req.Ville,
		Actif:        req.Actif,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (app *App) handleListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	services, err := app.ListServices(r.Context(), q.Get("categorie"), q.Get("ville"), q.Get("search"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (app *App) handleGetService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}

	svc, err := app.GetService(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (app *App) handleUpdateService(w http.ResponseWriter, r *http.Request) {
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

	var req updateServiceHTTPRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	svc, err := app.UpdateService(r.Context(), actorID, id, updateServiceRequestData{
		Titre:        req.Titre,
		Description:  req.Description,
		Categorie:    req.Categorie,
		DureeMinutes: req.DureeMinutes,
		Credits:      req.Credits,
		Ville:        req.Ville,
		Actif:        req.Actif,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (app *App) handleDeleteService(w http.ResponseWriter, r *http.Request) {
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

	if err := app.DeleteService(r.Context(), actorID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
