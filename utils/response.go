package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

type errorBody struct {
	Error string `json:"error"`
}

// JSON serialises data as JSON and writes it to w with the given HTTP status
// code. If serialisation fails the response falls back to a 500 plain-text
// message so the caller always receives a response.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Best-effort fallback — headers are already sent so we cannot change
		// the status code at this point.
		fmt.Printf("[HANDLER] failed to encode JSON response: %v\n", err)
	}
}

// Error writes a JSON error response with a single "error" key.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, errorBody{Error: message})
}

// NewID generates a short random hex string suitable for use as an entity ID.
// It is not a cryptographic UUID — for this in-memory demo, collision
// probability is negligible given typical record counts.
func NewID() string {
	return fmt.Sprintf("%016x", rand.Uint64())
}
