package main

import "net/http"

// NewRouter construit le mux HTTP avec middlewares et routes.
func NewRouter(svc *Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/users", svc.handleCreateUser)
	mux.HandleFunc("GET /api/users/{id}", svc.handleGetUser)
	mux.HandleFunc("PUT /api/users/{id}", svc.handleUpdateUser)
	mux.HandleFunc("GET /api/users/{id}/skills", svc.handleGetUserSkills)
	mux.HandleFunc("PUT /api/users/{id}/skills", svc.handleSetUserSkills)

	return chain(mux, recoveryMiddleware, loggingMiddleware, corsMiddleware)
}
