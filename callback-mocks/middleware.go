package main

import (
	"bytes"
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

var (
	mu             sync.Mutex
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
