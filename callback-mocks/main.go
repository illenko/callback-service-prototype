package main

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"time"
)

type CallbackResponse struct {
	Success bool `json:"success"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	http.Handle("/always-success", countMiddleware(http.HandlerFunc(alwaysSuccessHandler)))
	http.Handle("/success-delayed", countMiddleware(http.HandlerFunc(successDelayedHandler)))
	http.Handle("/always-fail", countMiddleware(http.HandlerFunc(alwaysFailHandler)))
	http.Handle("/random-fail", countMiddleware(http.HandlerFunc(randomFailHandler)))

	http.ListenAndServe(":8085", loggingMiddleware(http.DefaultServeMux))
}

const (
	errorRate   = 0.5
	contentType = "application/json"
)

func alwaysSuccessHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CallbackResponse{Success: true})
}

func successDelayedHandler(w http.ResponseWriter, _ *http.Request) {
	delay := time.Duration(3+rand.IntN(6)) * time.Second
	time.Sleep(delay)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CallbackResponse{Success: true})
}

func alwaysFailHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(ErrorResponse{Error: "Internal Server Error"})
}

func randomFailHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", contentType)
	if rand.Float64() < errorRate {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CallbackResponse{Success: true})
}
