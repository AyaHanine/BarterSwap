package main

// Service regroupe la logique métier (séparée de l'HTTP et du stockage).
type Service struct {
	store *Store
}

// NewService crée un Service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}
