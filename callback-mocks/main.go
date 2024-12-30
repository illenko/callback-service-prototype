package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
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

type loggingResponseWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request Headers: %v", r.Header)

		var requestBody bytes.Buffer
		tee := io.TeeReader(r.Body, &requestBody)
		body, err := io.ReadAll(tee)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
		}
		r.Body = io.NopCloser(&requestBody)
		log.Printf("Request Body: %s", body)

		lrw := &loggingResponseWriter{ResponseWriter: w, body: &bytes.Buffer{}}
		next.ServeHTTP(lrw, r)

		log.Printf("Response Headers: %v", w.Header())

		log.Printf("Response Body: %s", lrw.body.String())
	})
}

func main() {
	http.HandleFunc("/always-success", alwaysSuccessHandler)
	http.HandleFunc("/success-delayed", successDelayedHandler)
	http.HandleFunc("/always-fail", alwaysFailHandler)
	http.HandleFunc("/random-fail", randomFailHandler)

	// Wrap handlers with logging middleware
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
