package main

import "net/http"

// NewRouter construit le mux HTTP avec middlewares et routes.
func NewRouter(svc *Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Les routes utilisateurs seront branchées ici (gestion des utilisateurs).

	return chain(mux, recoveryMiddleware, loggingMiddleware, corsMiddleware)
}
