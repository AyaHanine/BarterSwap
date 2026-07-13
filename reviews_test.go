package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func completeExchangeWorkflow(t *testing.T, h http.Handler, owner, client User, svc Service) Exchange {
	t.Helper()

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	_, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	var ex Exchange
	if err := json.Unmarshal(body, &ex); err != nil {
		t.Fatalf("unmarshal exchange: %v", err)
	}

	code, got := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/accept", ex.ID), nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("accept status=%d body=%s", code, got)
	}
	code, got = doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/complete", ex.ID), nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", code, got)
	}
	if err := json.Unmarshal(got, &ex); err != nil {
		t.Fatalf("unmarshal completed exchange: %v", err)
	}
	return ex
}

func TestCreateReviewSuccess(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "rv_owner1", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "rv_client1", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille de haies", 2)
	ex := completeExchangeWorkflow(t, h, owner, client, svc)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	code, body := doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note":        5,
		"commentaire": "Excellent travail",
	}, clientAuth)
	if code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", code, body)
	}
	var rv Review
	if err := json.Unmarshal(body, &rv); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rv.Note != 5 || rv.AuthorID != client.ID || rv.TargetID != owner.ID || rv.ServiceID != svc.ID {
		t.Fatalf("review inattendu: %+v", rv)
	}
}

func TestCreateReviewWithoutAuth(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodPost, "/api/exchanges/1/review", map[string]any{
		"note": 4,
	}, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", code)
	}
}

func TestCreateReviewOnPendingExchange(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "rv_owner2", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "rv_client2", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	_, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	var ex Exchange
	_ = json.Unmarshal(body, &ex)

	code, _ := doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note": 3,
	}, clientAuth)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}

func TestCreateReviewInvalidNote(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "rv_owner3", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "rv_client3", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)
	ex := completeExchangeWorkflow(t, h, owner, client, svc)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	code, _ := doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note": 0,
	}, clientAuth)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}

func TestCreateReviewDuplicate(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "rv_owner4", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "rv_client4", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)
	ex := completeExchangeWorkflow(t, h, owner, client, svc)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	path := fmt.Sprintf("/api/exchanges/%d/review", ex.ID)
	_, _ = doJSON(t, h, http.MethodPost, path, map[string]any{"note": 4}, clientAuth)

	code, _ := doJSON(t, h, http.MethodPost, path, map[string]any{"note": 5}, clientAuth)
	if code != http.StatusConflict {
		t.Fatalf("status=%d, want 409", code)
	}
}

func TestCreateReviewForbiddenForOutsider(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "rv_owner5", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "rv_client5", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	outsider := createUserWithSkills(t, h, "rv_outsider5", []Skill{{Nom: "Sport", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)
	ex := completeExchangeWorkflow(t, h, owner, client, svc)

	outsiderAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", outsider.ID)}
	code, _ := doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note": 2,
	}, outsiderAuth)
	if code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", code)
	}
}

func TestListUserReviews(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "rv_owner6", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "rv_client6", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)
	ex := completeExchangeWorkflow(t, h, owner, client, svc)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	_, _ = doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note": 4,
	}, clientAuth)

	code, body := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/users/%d/reviews", owner.ID), nil, nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, body)
	}
	var reviews []Review
	if err := json.Unmarshal(body, &reviews); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(reviews) != 1 || reviews[0].TargetID != owner.ID {
		t.Fatalf("reviews=%+v", reviews)
	}
}

func TestListUserReviewsNotFound(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodGet, "/api/users/999/reviews", nil, nil)
	if code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", code)
	}
}

func TestListServiceReviews(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "rv_owner7", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "rv_client7", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)
	ex := completeExchangeWorkflow(t, h, owner, client, svc)

	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}
	_, _ = doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/exchanges/%d/review", ex.ID), map[string]any{
		"note": 5,
	}, ownerAuth)

	code, body := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/services/%d/reviews", svc.ID), nil, nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, body)
	}
	var reviews []Review
	if err := json.Unmarshal(body, &reviews); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(reviews) != 1 || reviews[0].ServiceID != svc.ID {
		t.Fatalf("reviews=%+v", reviews)
	}
}

func TestListServiceReviewsNotFound(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodGet, "/api/services/999/reviews", nil, nil)
	if code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", code)
	}
}
