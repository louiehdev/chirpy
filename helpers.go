package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
)

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type jsonError struct {
		Error string `json:"error"`
	}

	errData, err := json.Marshal(jsonError{Error: msg})
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(errData)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(jsonData)
}

func replaceProfane(s string) string {
	re := regexp.MustCompile(`kerfuffle|sharbert|fornax`)
	bodyString := strings.Fields(s)
	var cleanedBody []string

	for _, word := range bodyString {
		if re.MatchString(strings.ToLower(word)) {
			cleanedBody = append(cleanedBody, "****")
		} else {
			cleanedBody = append(cleanedBody, word)
		}
	}
	return strings.Join(cleanedBody, " ")
}
