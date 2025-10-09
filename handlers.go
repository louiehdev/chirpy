package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/louiehdev/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	hitResponse := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, cfg.fileserverHits.Load())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(hitResponse))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, cfg.platform)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(int32(0))
	cfg.db.DeleteUsers(r.Context())
}

func (cfg *apiConfig) healthHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) chirpHandler(w http.ResponseWriter, r *http.Request) {
	var params database.CreateChirpParams
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 500, "Something went wrong")
	} else if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
	} else if len(params.Body) <= 140 {
		params.Body = replaceProfane(params.Body)
		newChirp, err := cfg.db.CreateChirp(r.Context(), params)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
		}
		respondWithJSON(w, 201, newChirp)
	}
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 500, "Something went wrong")
	}
	newUser, err := cfg.db.CreateUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
	}

	respondWithJSON(w, 201, newUser)
}
