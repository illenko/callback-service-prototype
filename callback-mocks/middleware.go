package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

var (
	mu          sync.Mutex
	idMap       = make(map[string]bool)
	duplicateID = make(map[string]bool)
)

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

		// Parse the JSON payload to extract the ID
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err == nil {
			if id, ok := payload["id"].(string); ok {
				mu.Lock()
				if idMap[id] {
					duplicateID[id] = true
				} else {
					idMap[id] = true
				}
				mu.Unlock()
			}
		}

		lrw := &loggingResponseWriter{ResponseWriter: w, body: &bytes.Buffer{}}
		next.ServeHTTP(lrw, r)

		log.Printf("Response Headers: %v", w.Header())
		log.Printf("Response Body: %s", lrw.body.String())

		// Log duplicate IDs
		mu.Lock()
		for id := range duplicateID {
			log.Printf("Duplicate ID: %s", id)
		}
		mu.Unlock()
	})
}

var (
	endpointCounts = make(map[string]int)
)

func countMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		endpointCounts[r.URL.Path]++
		count := endpointCounts[r.URL.Path]
		mu.Unlock()

		log.Printf("Endpoint %s has been called %d times", r.URL.Path, count)
		next.ServeHTTP(w, r)
	})
}
