package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	currentHits := strconv.Itoa(int(cfg.fileserverHits.Load()))
	hitResponse := fmt.Sprintf("Hits: %s", currentHits)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(hitResponse))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(int32(0))
}

func main() {
	mux := http.NewServeMux()
	apiCfg := apiConfig{}
	handler := http.StripPrefix("/app", http.FileServer(http.Dir("")))

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	mux.HandleFunc("GET /healthz", httpHandler)
	mux.HandleFunc("GET /metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /reset", apiCfg.resetHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}

func httpHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
