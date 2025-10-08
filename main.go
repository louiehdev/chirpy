package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	apiCfg := apiConfig{}
	handler := http.StripPrefix("/app", http.FileServer(http.Dir("")))

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	mux.HandleFunc("GET /api/healthz", healthHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/validate_chirp", validationHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
