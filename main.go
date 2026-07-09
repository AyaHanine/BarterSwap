package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://barterswap:barterswap@localhost:5432/barterswap?sslmode=disable"
	}

	db, err := openDB(dbURL)
	if err != nil {
		log.Fatalf("connexion base de données: %v", err)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		log.Fatalf("migration: %v", err)
	}

	store := NewStore(db)
	svc := NewService(store)
	mux := NewRouter(svc)

	log.Printf("BarterSwap écoute sur %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("serveur HTTP: %v", err)
	}
}
