package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/louiehdev/chirpy/internal/auth"
	"github.com/louiehdev/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
	polkaKey       string
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

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := cfg.db.GetUserFromEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "incorrect email or password")
		return
	}
	if match, _ := auth.CheckPasswordHash(params.Password, user.HashedPassword); !match {
		respondWithError(w, 401, "incorrect email or password")
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	refreshToken, err := cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: auth.MakeRefreshToken(), ExpiresAt: time.Now().Add(1440 * time.Hour), UserID: user.ID})
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	userData := struct {
		Id           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		IsChirpyRed  bool      `json:"is_chirpy_red"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		IsChirpyRed:  user.IsChirpyRed,
		Token:        accessToken,
		RefreshToken: refreshToken.Token}

	respondWithJSON(w, 200, userData)
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	refreshToken, err := cfg.db.GetRefreshToken(r.Context(), token)
	if err != nil || time.Now().After(refreshToken.ExpiresAt) {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	newToken, err := auth.MakeJWT(refreshToken.UserID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	tokenData := struct {
		Token string `json:"token"`
	}{Token: newToken}
	respondWithJSON(w, 200, tokenData)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	refreshToken, err := cfg.db.GetRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	if err := cfg.db.RevokeToken(r.Context(), database.RevokeTokenParams{Token: refreshToken.Token, RevokedAt: sql.NullTime{Time: time.Now()}, UpdatedAt: time.Now()}); err != nil {
		respondWithError(w, 500, "Something went wrong")
		fmt.Println(err)
		return
	}
	respondWithError(w, 204, "Request Successful")
}

func (cfg *apiConfig) chirpHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	var params database.CreateChirpParams
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	} else if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	} else if len(params.Body) <= 140 {
		params.Body = replaceProfane(params.Body)
		params.UserID = userID
		newChirp, err := cfg.db.CreateChirp(r.Context(), params)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}
		respondWithJSON(w, 201, newChirp)
	}
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	hashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	userParams := database.CreateUserParams{Email: params.Email, HashedPassword: hashedPass}
	newUser, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	respondWithJSON(w, 201, newUser)
}

func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	var params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	hashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	userParams := database.UpdateUserParams{Email: params.Email, HashedPassword: hashedPass, ID: userID}
	updatedUser, err := cfg.db.UpdateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	respondWithJSON(w, 200, updatedUser)
}

func (cfg *apiConfig) upgradeUserHandler(w http.ResponseWriter, r *http.Request) {
	if key, err := auth.GetAPIKey(r.Header); err != nil || key != cfg.polkaKey {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	var params struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	if params.Event != "user.upgraded" {
		respondWithError(w, 204, "Unknown event")
		return
	}
	userID, _ := uuid.Parse(params.Data.UserID)
	if err := cfg.db.UpgradeUser(r.Context(), userID); err != nil {
		respondWithError(w, 404, "User not found")
		return
	}

	respondWithError(w, 204, "")
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) getChirpFromIDHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpID")
	idParam, _ := uuid.Parse(id)
	chirp, err := cfg.db.GetChirpFromID(r.Context(), idParam)
	if err != nil {
		respondWithError(w, 404, "Chirp not found")
		return
	}
	respondWithJSON(w, 200, chirp)
}

func (cfg *apiConfig) deleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	id := r.PathValue("chirpID")
	idParam, _ := uuid.Parse(id)
	chirp, err := cfg.db.GetChirpFromID(r.Context(), idParam)
	if err != nil {
		respondWithError(w, 404, "Chirp not found")
		return
	}
	if chirp.UserID != userID {
		respondWithError(w, 403, "Unauthorized")
		return
	}
	if err := cfg.db.DeleteChirp(r.Context(), chirp.ID); err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	respondWithError(w, 204, "Chirp deleted")
}
