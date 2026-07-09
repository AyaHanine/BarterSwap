package main

import "errors"

var (
	ErrNotFound     = errors.New("ressource introuvable")
	ErrConflict     = errors.New("conflit")
	ErrValidation   = errors.New("données invalides")
	ErrForbidden    = errors.New("accès interdit")
	ErrUnauthorized = errors.New("non authentifié")
)
