package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// MinimalRequest represents the minimal required fields
type Request struct {
	ActionType        string            `json:"actionType"`
	Event             Event             `json:"event"`
	AllowedOperations []Operation       `json:"allowedOperations,omitempty"`
}

// Event contains the event data
type Event struct {
	Request      RequestData   `json:"request"`
	AccessToken  AccessToken   `json:"accessToken"`
	RefreshToken *RefreshToken `json:"refreshToken,omitempty"`
}

// RequestData minimal request info
type RequestData struct {
	ClientID  string `json:"clientId"`
	GrantType string `json:"grantType"`
}

// AccessToken token info
type AccessToken struct {
	Scopes []string `json:"scopes"`
	Claims []Claim  `json:"claims"`
}

// RefreshToken refresh token info
type RefreshToken struct {
	Claims []Claim `json:"claims"`
}

// Claim represents a claim
type Claim struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// Operation allowed operations
type Operation struct {
	Op    string   `json:"op"`
	Paths []string `json:"paths"`
}

// Response represents the response to Asgardeo
type Response struct {
	Event Event `json:"event,omitempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Log minimal info
	log.Printf("Processing %s for client: %s", req.ActionType, req.Event.Request.ClientID)

	// For PoC: Just return success with the same event
	// In production, you'd modify claims/scopes here
	resp := Response{
		Event: req.Event,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// healthHandler provides a health check endpoint for Envoy gateway
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	http.HandleFunc("/token-validation", handler)
	// Health check endpoint for Envoy readiness probes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/healthz", healthHandler)
	
	// Explicitly bind to 0.0.0.0 to ensure Envoy can connect
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Extension service listening on %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
