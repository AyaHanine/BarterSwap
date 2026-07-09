package main

import (
	"context"
	"fmt"
	"strings"
)

var allowedSkillLevels = map[string]struct{}{
	"débutant":      {},
	"intermédiaire": {},
	"expert":        {},
}

// CreateUser crée un compte et attribue 10 crédits de bienvenue.
func (s *Service) CreateUser(ctx context.Context, pseudo, bio, ville string) (User, error) {
	pseudo = normalizePseudo(pseudo)
	if pseudo == "" {
		return User{}, fmt.Errorf("%w: pseudo obligatoire", ErrValidation)
	}
	return s.store.CreateUser(ctx, createUserInput{
		Pseudo: pseudo,
		Bio:    strings.TrimSpace(bio),
		Ville:  strings.TrimSpace(ville),
	})
}

// GetUser retourne le profil public d'un utilisateur.
func (s *Service) GetUser(ctx context.Context, id int) (User, error) {
	if id <= 0 {
		return User{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	return s.store.GetUser(ctx, id)
}

// UpdateUser modifie le profil de l'utilisateur authentifié.
func (s *Service) UpdateUser(ctx context.Context, actorID, userID int, pseudo, bio, ville *string) (User, error) {
	if userID <= 0 {
		return User{}, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if actorID <= 0 {
		return User{}, ErrUnauthorized
	}
	if actorID != userID {
		return User{}, ErrForbidden
	}

	in := updateUserInput{}
	if pseudo != nil {
		p := normalizePseudo(*pseudo)
		if p == "" {
			return User{}, fmt.Errorf("%w: pseudo obligatoire", ErrValidation)
		}
		in.Pseudo = &p
	}
	if bio != nil {
		b := strings.TrimSpace(*bio)
		in.Bio = &b
	}
	if ville != nil {
		v := strings.TrimSpace(*ville)
		in.Ville = &v
	}
	return s.store.UpdateUser(ctx, userID, in)
}

// GetUserSkills retourne les compétences d'un utilisateur.
func (s *Service) GetUserSkills(ctx context.Context, userID int) ([]Skill, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if _, err := s.store.GetUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.store.ListSkills(ctx, userID)
}

// SetUserSkills remplace entièrement les compétences d'un utilisateur.
func (s *Service) SetUserSkills(ctx context.Context, actorID, userID int, skills []Skill) ([]Skill, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: id invalide", ErrValidation)
	}
	if actorID <= 0 {
		return nil, ErrUnauthorized
	}
	if actorID != userID {
		return nil, ErrForbidden
	}
	if skills == nil {
		skills = []Skill{}
	}

	cleaned := make([]Skill, 0, len(skills))
	seen := make(map[string]struct{}, len(skills))
	for _, sk := range skills {
		nom := strings.TrimSpace(sk.Nom)
		niveau := strings.TrimSpace(sk.Niveau)
		if nom == "" {
			return nil, fmt.Errorf("%w: nom de compétence obligatoire", ErrValidation)
		}
		if _, ok := allowedSkillLevels[niveau]; !ok {
			return nil, fmt.Errorf("%w: niveau invalide (débutant, intermédiaire, expert)", ErrValidation)
		}
		key := strings.ToLower(nom)
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("%w: compétence en double", ErrValidation)
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, Skill{Nom: nom, Niveau: niveau})
	}
	return s.store.ReplaceSkills(ctx, userID, cleaned)
}
