package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://barterswap:barterswap@localhost:5435/barterswap?sslmode=disable"
	}
	return dsn
}

func setupTestAPI(t *testing.T) (*App, http.Handler, func()) {
	t.Helper()
	db, err := openDB(testDSN(t))
	if err != nil {
		t.Skipf("PostgreSQL indisponible: %v", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		t.Fatalf("migrate: %v", err)
	}
	cleanupTables(t, db)

	store := NewStore(db)
	app := NewApp(store)
	handler := NewRouter(app)

	cleanup := func() {
		cleanupTables(t, db)
		_ = db.Close()
	}
	return app, handler, cleanup
}

func cleanupTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
TRUNCATE credit_transactions, reviews, exchanges, skills, services, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, headers map[string]string) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, rr.Body.Bytes()
}

func TestCreateUserSuccess(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, body := doJSON(t, h, http.MethodPost, "/api/users", map[string]string{
		"pseudo": "alice",
		"bio":    "Jardinage",
		"ville":  "Lyon",
	}, nil)
	if code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", code, body)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if user.ID == 0 || user.Pseudo != "alice" {
		t.Fatalf("user inattendu: %+v", user)
	}
	if user.CreditBalance != welcomeCredits {
		t.Fatalf("crédits de bienvenue=%d, want %d", user.CreditBalance, welcomeCredits)
	}
	if user.CreatedAt == "" {
		t.Fatal("created_at manquant")
	}
}

func TestCreateUserEmptyPseudo(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	tests := []struct {
		name string
		body map[string]string
	}{
		{name: "pseudo vide", body: map[string]string{"pseudo": ""}},
		{name: "pseudo espaces", body: map[string]string{"pseudo": "   "}},
		{name: "pseudo absent", body: map[string]string{"bio": "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _ := doJSON(t, h, http.MethodPost, "/api/users", tt.body, nil)
			if code != http.StatusBadRequest {
				t.Fatalf("status=%d, want 400", code)
			}
		})
	}
}

func TestCreateUserDuplicatePseudo(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	code, _ := doJSON(t, h, http.MethodPost, "/api/users", map[string]string{"pseudo": "bob"}, nil)
	if code != http.StatusCreated {
		t.Fatalf("premier create status=%d", code)
	}
	code, _ = doJSON(t, h, http.MethodPost, "/api/users", map[string]string{"pseudo": "bob"}, nil)
	if code != http.StatusConflict {
		t.Fatalf("status=%d, want 409", code)
	}
}

func TestGetUser(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	_, body := doJSON(t, h, http.MethodPost, "/api/users", map[string]string{"pseudo": "carol"}, nil)
	var created User
	_ = json.Unmarshal(body, &created)

	code, got := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/users/%d", created.ID), nil, nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, got)
	}

	code, _ = doJSON(t, h, http.MethodGet, "/api/users/99999", nil, nil)
	if code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", code)
	}
}

func TestUpdateUser(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	_, body := doJSON(t, h, http.MethodPost, "/api/users", map[string]string{"pseudo": "dave"}, nil)
	var created User
	_ = json.Unmarshal(body, &created)

	t.Run("sans auth", func(t *testing.T) {
		code, _ := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/users/%d", created.ID), map[string]string{
			"bio": "nouvelle bio",
		}, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", code)
		}
	})

	t.Run("autre utilisateur", func(t *testing.T) {
		_, otherBody := doJSON(t, h, http.MethodPost, "/api/users", map[string]string{"pseudo": "eve"}, nil)
		var other User
		_ = json.Unmarshal(otherBody, &other)
		code, _ := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/users/%d", created.ID), map[string]string{
			"bio": "hack",
		}, map[string]string{"X-User-ID": fmt.Sprintf("%d", other.ID)})
		if code != http.StatusForbidden {
			t.Fatalf("status=%d, want 403", code)
		}
	})

	t.Run("succès", func(t *testing.T) {
		code, got := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/users/%d", created.ID), map[string]any{
			"bio":   "Passionné de bricolage",
			"ville": "Nantes",
		}, map[string]string{"X-User-ID": fmt.Sprintf("%d", created.ID)})
		if code != http.StatusOK {
			t.Fatalf("status=%d body=%s", code, got)
		}
		var user User
		_ = json.Unmarshal(got, &user)
		if user.Bio != "Passionné de bricolage" || user.Ville != "Nantes" {
			t.Fatalf("profil non mis à jour: %+v", user)
		}
	})

	t.Run("pseudo vide", func(t *testing.T) {
		code, _ := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/users/%d", created.ID), map[string]string{
			"pseudo": "",
		}, map[string]string{"X-User-ID": fmt.Sprintf("%d", created.ID)})
		if code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", code)
		}
	})
}

func TestUserSkills(t *testing.T) {
	_, h, cleanup := setupTestAPI(t)
	defer cleanup()

	_, body := doJSON(t, h, http.MethodPost, "/api/users", map[string]string{"pseudo": "frank"}, nil)
	var created User
	_ = json.Unmarshal(body, &created)
	auth := map[string]string{"X-User-ID": fmt.Sprintf("%d", created.ID)}
	path := fmt.Sprintf("/api/users/%d/skills", created.ID)

	t.Run("get vide", func(t *testing.T) {
		code, got := doJSON(t, h, http.MethodGet, path, nil, nil)
		if code != http.StatusOK {
			t.Fatalf("status=%d", code)
		}
		var skills []Skill
		_ = json.Unmarshal(got, &skills)
		if len(skills) != 0 {
			t.Fatalf("skills=%v, want []", skills)
		}
	})

	t.Run("put invalide niveau", func(t *testing.T) {
		code, _ := doJSON(t, h, http.MethodPut, path, map[string]any{
			"skills": []Skill{{Nom: "Jardinage", Niveau: "dieu"}},
		}, auth)
		if code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", code)
		}
	})

	t.Run("put succès et écrasement", func(t *testing.T) {
		code, got := doJSON(t, h, http.MethodPut, path, map[string]any{
			"skills": []Skill{
				{Nom: "Jardinage", Niveau: "expert"},
				{Nom: "Cuisine", Niveau: "débutant"},
			},
		}, auth)
		if code != http.StatusOK {
			t.Fatalf("status=%d body=%s", code, got)
		}
		var skills []Skill
		_ = json.Unmarshal(got, &skills)
		if len(skills) != 2 {
			t.Fatalf("len=%d, want 2", len(skills))
		}

		code, got = doJSON(t, h, http.MethodPut, path, map[string]any{
			"skills": []Skill{{Nom: "Musique", Niveau: "intermédiaire"}},
		}, auth)
		if code != http.StatusOK {
			t.Fatalf("status=%d body=%s", code, got)
		}
		_ = json.Unmarshal(got, &skills)
		if len(skills) != 1 || skills[0].Nom != "Musique" {
			t.Fatalf("écrasement attendu, got %+v", skills)
		}

		code, profileBody := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/users/%d", created.ID), nil, nil)
		if code != http.StatusOK {
			t.Fatalf("get user status=%d", code)
		}
		var user User
		_ = json.Unmarshal(profileBody, &user)
		if len(user.Skills) != 1 || user.Skills[0].Nom != "Musique" {
			t.Fatalf("skills dans profil: %+v", user.Skills)
		}
	})

	t.Run("put autre utilisateur", func(t *testing.T) {
		code, _ := doJSON(t, h, http.MethodPut, path, map[string]any{
			"skills": []Skill{{Nom: "Sport", Niveau: "débutant"}},
		}, map[string]string{"X-User-ID": "999"})
		if code != http.StatusForbidden {
			t.Fatalf("status=%d, want 403", code)
		}
	})
}

func TestServiceCreateUserValidation(t *testing.T) {
	svc := NewApp(nil)
	ctx := context.Background()
	_, err := svc.CreateUser(ctx, "  ", "", "")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err=%v, want ErrValidation", err)
	}
}

func TestHealth(t *testing.T) {
	h := NewRouter(NewApp(NewStore(nil)))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestWelcomeCreditsTransaction(t *testing.T) {
	svc, _, cleanup := setupTestAPI(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := svc.CreateUser(ctx, "grace", "", "Paris")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if user.CreditBalance != 10 {
		t.Fatalf("balance=%d", user.CreditBalance)
	}

	var count int
	err = svc.store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM credit_transactions WHERE user_id=$1 AND type='earn' AND montant=10`,
		user.ID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query tx: %v", err)
	}
	if count != 1 {
		t.Fatalf("transactions=%d, want 1", count)
	}
}
