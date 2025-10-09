package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/louiehdev/chirpy/internal/database"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	cfg := apiConfig{db: dbQueries, platform: platform}

	mux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir("")))

	mux.Handle("/app/", cfg.middlewareMetricsInc(handler))
	mux.HandleFunc("GET /api/healthz", cfg.healthHandler)
	mux.HandleFunc("GET /admin/metrics", cfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.resetHandler)
	mux.HandleFunc("POST /api/chirps", cfg.chirpHandler)
	mux.HandleFunc("POST /api/users", cfg.createUserHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
