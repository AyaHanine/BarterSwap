package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"
)

func TestGetUserStatsNotFound(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodGet, "/api/users/999/stats", nil, nil)
	if code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", code)
	}
}

func TestGetUserStatsInvalidID(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodGet, "/api/users/abc/stats", nil, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}

func TestGetUserStatsFreshUser(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	user := createUserWithSkills(t, h, "stats_fresh", []Skill{
		{Nom: "Jardinage", Niveau: "débutant"},
	})

	code, body := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/users/%d/stats", user.ID), nil, nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, body)
	}

	var stats UserStats
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if stats.UserID != user.ID {
		t.Fatalf("user_id=%d, want %d", stats.UserID, user.ID)
	}
	if stats.CreditBalance != 10 {
		t.Fatalf("credit_balance=%d, want 10", stats.CreditBalance)
	}
	if stats.ServicesActifs != 0 || stats.EchangesCompletes != 0 || stats.NbAvis != 0 {
		t.Fatalf("stats inattendues pour utilisateur neuf: %+v", stats)
	}
	if stats.NoteMoyenne != 0 || stats.TotalGagne != 0 || stats.TotalDepense != 0 {
		t.Fatalf("totaux inattendus: %+v", stats)
	}
}

func TestGetUserStatsAfterCompletedExchange(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "stats_owner", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "stats_client", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille de haies", 2)
	ex := completeExchangeWorkflow(t, h, owner, client, svc)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	code, body := doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note":        4,
		"commentaire": "Bien",
	}, clientAuth)
	if code != http.StatusCreated {
		t.Fatalf("client review status=%d body=%s", code, body)
	}
	code, body = doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note": 5,
	}, ownerAuth)
	if code != http.StatusCreated {
		t.Fatalf("owner review status=%d body=%s", code, body)
	}

	code, body = doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/users/%d/stats", owner.ID), nil, nil)
	if code != http.StatusOK {
		t.Fatalf("owner stats status=%d body=%s", code, body)
	}
	var ownerStats UserStats
	if err := json.Unmarshal(body, &ownerStats); err != nil {
		t.Fatalf("unmarshal owner stats: %v", err)
	}
	if ownerStats.ServicesActifs != 1 {
		t.Fatalf("services_actifs=%d, want 1", ownerStats.ServicesActifs)
	}
	if ownerStats.EchangesCompletes != 1 {
		t.Fatalf("echanges_completes=%d, want 1", ownerStats.EchangesCompletes)
	}
	if ownerStats.CreditBalance != 12 { // 10 bienvenue + 2 gagnés
		t.Fatalf("credit_balance=%d, want 12", ownerStats.CreditBalance)
	}
	if ownerStats.NbAvis != 1 || math.Abs(ownerStats.NoteMoyenne-4) > 0.01 {
		t.Fatalf("avis owner: nb=%d moyenne=%v, want 1 / 4", ownerStats.NbAvis, ownerStats.NoteMoyenne)
	}
	if ownerStats.TotalGagne != 2 {
		t.Fatalf("total_gagne=%d, want 2", ownerStats.TotalGagne)
	}
	if ownerStats.TotalDepense != 0 {
		t.Fatalf("total_depense=%d, want 0", ownerStats.TotalDepense)
	}

	code, body = doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/users/%d/stats", client.ID), nil, nil)
	if code != http.StatusOK {
		t.Fatalf("client stats status=%d body=%s", code, body)
	}
	var clientStats UserStats
	if err := json.Unmarshal(body, &clientStats); err != nil {
		t.Fatalf("unmarshal client stats: %v", err)
	}
	if clientStats.EchangesCompletes != 1 {
		t.Fatalf("client echanges_completes=%d, want 1", clientStats.EchangesCompletes)
	}
	if clientStats.CreditBalance != 8 { // 10 − 2
		t.Fatalf("client credit_balance=%d, want 8", clientStats.CreditBalance)
	}
	if clientStats.NbAvis != 1 || math.Abs(clientStats.NoteMoyenne-5) > 0.01 {
		t.Fatalf("avis client: nb=%d moyenne=%v, want 1 / 5", clientStats.NbAvis, clientStats.NoteMoyenne)
	}
	if clientStats.TotalGagne != 0 {
		t.Fatalf("client total_gagne=%d, want 0", clientStats.TotalGagne)
	}
	if clientStats.TotalDepense != 2 {
		t.Fatalf("client total_depense=%d, want 2", clientStats.TotalDepense)
	}
}
