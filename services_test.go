package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func createUserWithSkills(t *testing.T, h http.Handler, pseudo string, skills []Skill) User {
	t.Helper()
	code, body := doJSON(t, h, http.MethodPost, "/api/users", map[string]string{
		"pseudo": pseudo,
		"ville":  "Lyon",
	}, nil)
	if code != http.StatusCreated {
		t.Fatalf("create user status=%d body=%s", code, body)
	}
	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatalf("unmarshal user: %v", err)
	}
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", user.ID)}
	code, body = doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/users/%d/skills", user.ID), map[string]any{
		"skills": skills,
	}, auth)
	if code != http.StatusOK {
		t.Fatalf("set skills status=%d body=%s", code, body)
	}
	return user
}

func TestCreateServiceSuccess(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	user := createUserWithSkills(t, h, "provider1", []Skill{
		{Nom: "Jardinage", Niveau: "expert"},
	})
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", user.ID)}

	code, body := doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre":         "Taille de haies",
		"description":   "Je taille vos haies",
		"categorie":     "Jardinage",
		"duree_minutes": 60,
		"credits":       2,
		"ville":         "Lyon",
	}, auth)
	if code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", code, body)
	}
	var svc Service
	if err := json.Unmarshal(body, &svc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if svc.ID == 0 || svc.ProviderID != user.ID || !svc.Actif {
		t.Fatalf("service inattendu: %+v", svc)
	}
	if svc.Titre != "Taille de haies" || svc.Credits != 2 {
		t.Fatalf("champs incorrects: %+v", svc)
	}
}

func TestCreateServiceWithoutAuth(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre": "x", "categorie": "Jardinage", "duree_minutes": 30, "credits": 1,
	}, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", code)
	}
}

func TestCreateServiceWithoutSkill(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	user := createUserWithSkills(t, h, "provider2", []Skill{
		{Nom: "Cuisine", Niveau: "débutant"},
	})
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", user.ID)}

	code, _ := doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre":         "Taille de haies",
		"categorie":     "Jardinage",
		"duree_minutes": 60,
		"credits":       2,
	}, auth)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}

func TestCreateServiceValidation(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	user := createUserWithSkills(t, h, "provider3", []Skill{
		{Nom: "Jardinage", Niveau: "expert"},
		{Nom: "Autre", Niveau: "débutant"},
	})
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", user.ID)}

	tests := []struct {
		name string
		body map[string]any
	}{
		{name: "titre vide", body: map[string]any{"titre": "", "categorie": "Jardinage", "duree_minutes": 30, "credits": 1}},
		{name: "catégorie invalide", body: map[string]any{"titre": "x", "categorie": "Magie", "duree_minutes": 30, "credits": 1}},
		{name: "durée invalide", body: map[string]any{"titre": "x", "categorie": "Jardinage", "duree_minutes": 0, "credits": 1}},
		{name: "crédits invalides", body: map[string]any{"titre": "x", "categorie": "Jardinage", "duree_minutes": 30, "credits": 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _ := doJSON(t, h, http.MethodPost, "/api/services", tt.body, auth)
			if code != http.StatusBadRequest {
				t.Fatalf("status=%d, want 400", code)
			}
		})
	}
}

func TestGetAndDeleteService(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	user := createUserWithSkills(t, h, "provider4", []Skill{{Nom: "Musique", Niveau: "intermédiaire"}})
	other := createUserWithSkills(t, h, "other4", []Skill{{Nom: "Musique", Niveau: "débutant"}})
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", user.ID)}

	_, body := doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre": "Cours de piano", "categorie": "Musique", "duree_minutes": 45, "credits": 3, "ville": "Paris",
	}, auth)
	var created Service
	_ = json.Unmarshal(body, &created)

	code, got := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/services/%d", created.ID), nil, nil)
	if code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", code, got)
	}

	code, _ = doJSON(t, h, http.MethodGet, "/api/services/99999", nil, nil)
	if code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", code)
	}

	code, _ = doJSON(t, h, http.MethodDelete, fmt.Sprintf("/api/services/%d", created.ID), nil, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("delete sans auth status=%d, want 401", code)
	}

	code, _ = doJSON(t, h, http.MethodDelete, fmt.Sprintf("/api/services/%d", created.ID), nil, map[string]string{
		"X-User-ID": fmt.Sprintf("%d", other.ID),
	})
	if code != http.StatusForbidden {
		t.Fatalf("delete autre user status=%d, want 403", code)
	}

	code, _ = doJSON(t, h, http.MethodDelete, fmt.Sprintf("/api/services/%d", created.ID), nil, auth)
	if code != http.StatusNoContent {
		t.Fatalf("delete status=%d, want 204", code)
	}

	code, _ = doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/services/%d", created.ID), nil, nil)
	if code != http.StatusNotFound {
		t.Fatalf("après delete status=%d, want 404", code)
	}
}

func TestUpdateService(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	user := createUserWithSkills(t, h, "provider5", []Skill{
		{Nom: "Bricolage", Niveau: "expert"},
		{Nom: "Cuisine", Niveau: "débutant"},
	})
	other := createUserWithSkills(t, h, "other5", []Skill{{Nom: "Bricolage", Niveau: "débutant"}})
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", user.ID)}

	_, body := doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre": "Montage meuble", "categorie": "Bricolage", "duree_minutes": 90, "credits": 4, "ville": "Nantes",
	}, auth)
	var created Service
	_ = json.Unmarshal(body, &created)

	code, _ := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/services/%d", created.ID), map[string]any{
		"titre": "Montage meuble IKEA",
	}, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", code)
	}

	code, _ = doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/services/%d", created.ID), map[string]any{
		"titre": "hack",
	}, map[string]string{"X-User-ID": fmt.Sprintf("%d", other.ID)})
	if code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", code)
	}

	code, got := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/services/%d", created.ID), map[string]any{
		"titre":         "Montage meuble IKEA",
		"credits":       5,
		"actif":         false,
		"categorie":     "Cuisine",
		"duree_minutes": 120,
	}, auth)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, got)
	}
	var updated Service
	_ = json.Unmarshal(got, &updated)
	if updated.Titre != "Montage meuble IKEA" || updated.Credits != 5 || updated.Actif || updated.Categorie != "Cuisine" {
		t.Fatalf("update incorrect: %+v", updated)
	}

	code, _ = doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/services/%d", created.ID), map[string]any{
		"categorie": "Informatique",
	}, auth)
	if code != http.StatusBadRequest {
		t.Fatalf("catégorie sans skill status=%d, want 400", code)
	}
}

func TestListServicesFilters(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	u1 := createUserWithSkills(t, h, "filter1", []Skill{
		{Nom: "Jardinage", Niveau: "expert"},
		{Nom: "Cuisine", Niveau: "débutant"},
	})
	u2 := createUserWithSkills(t, h, "filter2", []Skill{
		{Nom: "Sport", Niveau: "intermédiaire"},
	})
	a1 := map[string]string{"X-User-ID": fmt.Sprintf("%d", u1.ID)}
	a2 := map[string]string{"X-User-ID": fmt.Sprintf("%d", u2.ID)}

	doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre": "Taille de haies", "description": "jardin soigné", "categorie": "Jardinage",
		"duree_minutes": 60, "credits": 2, "ville": "Lyon",
	}, a1)
	doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre": "Cours de pâtisserie", "description": "gâteaux", "categorie": "Cuisine",
		"duree_minutes": 90, "credits": 3, "ville": "Paris",
	}, a1)
	doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre": "Coaching running", "description": "préparation 10km", "categorie": "Sport",
		"duree_minutes": 45, "credits": 2, "ville": "Lyon",
	}, a2)

	code, body := doJSON(t, h, http.MethodGet, "/api/services", nil, nil)
	if code != http.StatusOK {
		t.Fatalf("list status=%d", code)
	}
	var all []Service
	_ = json.Unmarshal(body, &all)
	if len(all) != 3 {
		t.Fatalf("len=%d, want 3", len(all))
	}

	code, body = doJSON(t, h, http.MethodGet, "/api/services?categorie=Jardinage", nil, nil)
	if code != http.StatusOK {
		t.Fatalf("filter categorie status=%d", code)
	}
	var byCat []Service
	_ = json.Unmarshal(body, &byCat)
	if len(byCat) != 1 || byCat[0].Categorie != "Jardinage" {
		t.Fatalf("filtre categorie: %+v", byCat)
	}

	code, body = doJSON(t, h, http.MethodGet, "/api/services?ville=Lyon", nil, nil)
	if code != http.StatusOK {
		t.Fatalf("filter ville status=%d", code)
	}
	var byVille []Service
	_ = json.Unmarshal(body, &byVille)
	if len(byVille) != 2 {
		t.Fatalf("filtre ville len=%d, want 2", len(byVille))
	}

	code, body = doJSON(t, h, http.MethodGet, "/api/services?search=pâtisserie", nil, nil)
	if code != http.StatusOK {
		t.Fatalf("search status=%d", code)
	}
	var bySearch []Service
	_ = json.Unmarshal(body, &bySearch)
	if len(bySearch) != 1 || bySearch[0].Titre != "Cours de pâtisserie" {
		t.Fatalf("search: %+v", bySearch)
	}

	code, body = doJSON(t, h, http.MethodGet, "/api/services?categorie=Sport&ville=Lyon", nil, nil)
	if code != http.StatusOK {
		t.Fatalf("multi filter status=%d", code)
	}
	var multi []Service
	_ = json.Unmarshal(body, &multi)
	if len(multi) != 1 || multi[0].Titre != "Coaching running" {
		t.Fatalf("multi filter: %+v", multi)
	}
}
