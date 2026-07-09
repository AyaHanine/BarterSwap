package main

// App regroupe la logique métier (séparée de l'HTTP et du stockage).
type App struct {
	store *Store
}

// NewApp crée un App.
func NewApp(store *Store) *App {
	return &App{store: store}
}
