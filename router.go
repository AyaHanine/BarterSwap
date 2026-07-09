package main

import "net/http"

// NewRouter construit le mux HTTP avec middlewares et routes.
func NewRouter(app *App) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/users", app.handleCreateUser)
	mux.HandleFunc("GET /api/users/{id}", app.handleGetUser)
	mux.HandleFunc("PUT /api/users/{id}", app.handleUpdateUser)
	mux.HandleFunc("GET /api/users/{id}/skills", app.handleGetUserSkills)
	mux.HandleFunc("PUT /api/users/{id}/skills", app.handleSetUserSkills)

	mux.HandleFunc("GET /api/services", app.handleListServices)
	mux.HandleFunc("POST /api/services", app.handleCreateService)
	mux.HandleFunc("GET /api/services/{id}", app.handleGetService)
	mux.HandleFunc("PUT /api/services/{id}", app.handleUpdateService)
	mux.HandleFunc("DELETE /api/services/{id}", app.handleDeleteService)

	return chain(mux, recoveryMiddleware, loggingMiddleware, corsMiddleware)
}
