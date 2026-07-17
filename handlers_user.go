package main

import (
	"errors"
	"net/http"
	"strconv"
)

type createUserRequest struct {
	Pseudo string `json:"pseudo"`
	Bio    string `json:"bio"`
	Ville  string `json:"ville"`
}

type updateUserRequest struct {
	Pseudo *string `json:"pseudo"`
	Bio    *string `json:"bio"`
	Ville  *string `json:"ville"`
}

type setSkillsRequest struct {
	Skills []Skill `json:"skills"`
}

func (app *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	user, err := app.CreateUser(r.Context(), req.Pseudo, req.Bio, req.Ville)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (app *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}

	user, err := app.GetUser(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (app *App) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
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

	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	user, err := app.UpdateUser(r.Context(), actorID, id, req.Pseudo, req.Bio, req.Ville)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (app *App) handleGetUserSkills(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}

	skills, err := app.GetUserSkills(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (app *App) handleSetUserSkills(w http.ResponseWriter, r *http.Request) {
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

	var req setSkillsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	skills, err := app.SetUserSkills(r.Context(), actorID, id, req.Skills)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (app *App) handleGetUserStats(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}

	stats, err := app.GetUserStats(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func pathID(r *http.Request, name string) (int, error) {
	return strconv.Atoi(r.PathValue(name))
}

func userIDFromHeader(r *http.Request) (int, error) {
	raw := r.Header.Get("X-User-ID")
	if raw == "" {
		return 0, ErrUnauthorized
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, ErrUnauthorized
	}
	return id, nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}
