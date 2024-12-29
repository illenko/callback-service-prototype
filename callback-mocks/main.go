package main

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
)

type CallbackResponse struct {
	Success bool `json:"success"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	http.HandleFunc("/always-success", alwaysSuccessHandler)
	http.HandleFunc("/always-fail", alwaysFailHandler)
	http.HandleFunc("/random-fail", randomFailHandler)

	http.ListenAndServe(":8085", nil)
}

const (
	errorRate   = 0.5
	contentType = "application/json"
)

func alwaysSuccessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CallbackResponse{Success: true})
}

func alwaysFailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(ErrorResponse{Error: "Internal Server Error"})
}

func randomFailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentType)
	if rand.Float64() < errorRate {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CallbackResponse{Success: true})
}
