package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func createServiceForUser(t *testing.T, h http.Handler, user User, categorie, titre string, credits int) Service {
	t.Helper()
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", user.ID)}
	code, body := doJSON(t, h, http.MethodPost, "/api/services", map[string]any{
		"titre":         titre,
		"categorie":     categorie,
		"duree_minutes": 60,
		"credits":       credits,
		"ville":         "Lyon",
	}, auth)
	if code != http.StatusCreated {
		t.Fatalf("create service status=%d body=%s", code, body)
	}
	var svc Service
	if err := json.Unmarshal(body, &svc); err != nil {
		t.Fatalf("unmarshal service: %v", err)
	}
	return svc
}

func TestCreateExchangeSuccess(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner1", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client1", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille de haies", 2)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	code, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	if code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", code, body)
	}
	var ex Exchange
	if err := json.Unmarshal(body, &ex); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ex.Status != exchangeStatusPending || ex.RequesterID != client.ID || ex.OwnerID != owner.ID {
		t.Fatalf("exchange inattendu: %+v", ex)
	}
}

func TestCreateExchangeWithoutAuth(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": 1,
	}, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", code)
	}
}

func TestCreateExchangeOwnService(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner2", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	code, _ := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, auth)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}

func TestCreateExchangeInsufficientCredits(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner3", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client3", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 15)
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}

	code, _ := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, auth)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}

func TestExchangeWorkflowAndCredits(t *testing.T) {
	app, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner4", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client4", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 3)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	_, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	var ex Exchange
	_ = json.Unmarshal(body, &ex)

	code, got := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/accept", ex.ID), nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("accept status=%d body=%s", code, got)
	}

	clientAfterAccept, _ := app.GetUser(context.Background(), client.ID)
	ownerAfterAccept, _ := app.GetUser(context.Background(), owner.ID)
	if clientAfterAccept.CreditBalance != 7 {
		t.Fatalf("client after accept=%d, want 7", clientAfterAccept.CreditBalance)
	}
	if ownerAfterAccept.CreditBalance != 10 {
		t.Fatalf("owner after accept=%d, want 10", ownerAfterAccept.CreditBalance)
	}

	code, got = doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/complete", ex.ID), nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", code, got)
	}
	var completed Exchange
	_ = json.Unmarshal(got, &completed)
	if completed.Status != exchangeStatusCompleted {
		t.Fatalf("status=%q, want completed", completed.Status)
	}

	clientUser, _ := app.GetUser(context.Background(), client.ID)
	if clientUser.CreditBalance != 7 {
		t.Fatalf("client balance=%d, want 7", clientUser.CreditBalance)
	}
	ownerUser, _ := app.GetUser(context.Background(), owner.ID)
	if ownerUser.CreditBalance != 13 {
		t.Fatalf("owner balance=%d, want 13", ownerUser.CreditBalance)
	}
}

func TestAcceptExchangeForbiddenForRequester(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner5", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client5", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	_, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	var ex Exchange
	_ = json.Unmarshal(body, &ex)

	code, _ := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/accept", ex.ID), nil, clientAuth)
	if code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", code)
	}
}

func TestRejectExchange(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner6", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client6", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	_, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	var ex Exchange
	_ = json.Unmarshal(body, &ex)

	code, got := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/reject", ex.ID), nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("reject status=%d body=%s", code, got)
	}
	var rejected Exchange
	_ = json.Unmarshal(got, &rejected)
	if rejected.Status != exchangeStatusRejected {
		t.Fatalf("status=%q, want rejected", rejected.Status)
	}
}

func TestCancelExchange(t *testing.T) {
	app, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner7", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client7", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	_, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	var ex Exchange
	_ = json.Unmarshal(body, &ex)

	code, got := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/cancel", ex.ID), nil, clientAuth)
	if code != http.StatusOK {
		t.Fatalf("cancel pending status=%d body=%s", code, got)
	}

	_, body = doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	_ = json.Unmarshal(body, &ex)

	code, got = doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/accept", ex.ID), nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("accept status=%d body=%s", code, got)
	}

	code, got = doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/cancel", ex.ID), nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("cancel accepted status=%d body=%s", code, got)
	}

	clientUser, _ := app.GetUser(context.Background(), client.ID)
	if clientUser.CreditBalance != 10 {
		t.Fatalf("client after cancel=%d, want 10 (crédits restitués)", clientUser.CreditBalance)
	}
}

func TestListExchanges(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner8", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client8", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{"service_id": svc.ID}, clientAuth)

	code, body := doJSON(t, h, http.MethodGet, "/api/exchanges", nil, clientAuth)
	if code != http.StatusOK {
		t.Fatalf("list client status=%d", code)
	}
	var clientList []Exchange
	_ = json.Unmarshal(body, &clientList)
	if len(clientList) != 1 {
		t.Fatalf("client list len=%d, want 1", len(clientList))
	}

	code, body = doJSON(t, h, http.MethodGet, "/api/exchanges", nil, ownerAuth)
	if code != http.StatusOK {
		t.Fatalf("list owner status=%d", code)
	}
	var ownerList []Exchange
	_ = json.Unmarshal(body, &ownerList)
	if len(ownerList) != 1 {
		t.Fatalf("owner list len=%d, want 1", len(ownerList))
	}

	code, body = doJSON(t, h, http.MethodGet, "/api/exchanges?status=pending", nil, clientAuth)
	if code != http.StatusOK {
		t.Fatalf("filter status=%d", code)
	}
	var pending []Exchange
	_ = json.Unmarshal(body, &pending)
	if len(pending) != 1 || pending[0].Status != exchangeStatusPending {
		t.Fatalf("pending filter: %+v", pending)
	}
}

func TestGetExchangeNotFound(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodGet, "/api/exchanges/99999", nil, nil)
	if code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", code)
	}
}

func TestCompleteExchangeRequiresAccepted(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner9", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client := createUserWithSkills(t, h, "ex_client9", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)

	clientAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", client.ID)}
	ownerAuth := map[string]string{"X-User-ID": fmt.Sprintf("%d", owner.ID)}

	_, body := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{
		"service_id": svc.ID,
	}, clientAuth)
	var ex Exchange
	_ = json.Unmarshal(body, &ex)

	code, _ := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/exchanges/%d/complete", ex.ID), nil, ownerAuth)
	if code != http.StatusConflict {
		t.Fatalf("complete pending status=%d, want 409", code)
	}
}

func TestServiceAlreadyReserved(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	owner := createUserWithSkills(t, h, "ex_owner10", []Skill{{Nom: "Jardinage", Niveau: "expert"}})
	client1 := createUserWithSkills(t, h, "ex_client10a", []Skill{{Nom: "Cuisine", Niveau: "débutant"}})
	client2 := createUserWithSkills(t, h, "ex_client10b", []Skill{{Nom: "Musique", Niveau: "débutant"}})
	svc := createServiceForUser(t, h, owner, "Jardinage", "Taille", 2)

	auth1 := map[string]string{"X-User-ID": fmt.Sprintf("%d", client1.ID)}
	auth2 := map[string]string{"X-User-ID": fmt.Sprintf("%d", client2.ID)}

	doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{"service_id": svc.ID}, auth1)
	code, _ := doJSON(t, h, http.MethodPost, "/api/exchanges", map[string]any{"service_id": svc.ID}, auth2)
	if code != http.StatusConflict {
		t.Fatalf("status=%d, want 409", code)
	}
}
