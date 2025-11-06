package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

// Header represents a header in additionalHeaders
type Header struct {
	Name  string   `json:"name"`
	Value []string `json:"value"`
}

// RequestData minimal request info
type RequestData struct {
	ClientID          string   `json:"clientId"`
	GrantType         string   `json:"grantType"`
	AdditionalHeaders []Header `json:"additionalHeaders,omitempty"`
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
	ActionStatus        string             `json:"actionStatus"`
	Operations          []OperationResponse `json:"operations,omitempty"`
	FailureReason      string             `json:"failureReason,omitempty"`
	FailureDescription string             `json:"failureDescription,omitempty"`
	ErrorMessage       string             `json:"errorMessage,omitempty"`
	ErrorDescription   string             `json:"errorDescription,omitempty"`
}

// OperationResponse represents an operation in the response
type OperationResponse struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// EntitlementsData represents the structure of entitlements.json
type EntitlementsData struct {
	Entitlements []Entitlement `json:"entitlements"`
}

// Entitlement represents a single entitlement
type Entitlement struct {
	EntitlementID string                 `json:"entitlementId"`
	Subject       Subject                 `json:"subject"`
	Action        string                  `json:"action"`
	Object        map[string]interface{} `json:"object"`
	Constraints   map[string]interface{} `json:"constraints"`
}

// Subject represents the subject in an entitlement
type Subject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log full request details
	log.Printf("=== Full Request Details ===")
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Protocol: %s", r.Proto)
	log.Printf("Remote Address: %s", r.RemoteAddr)
	
	// Log all headers
	log.Printf("Headers:")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	// Read and log body
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// Log body as string
	log.Printf("Body: %s", string(bodyBytes))
	log.Printf("=== End Request Details ===")

	// Restore body for decoding
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Log parsed request info
	log.Printf("Processing %s for client: %s", req.ActionType, req.Event.Request.ClientID)
	
	// Log AdditionalHeaders if present
	if len(req.Event.Request.AdditionalHeaders) > 0 {
		log.Printf("AdditionalHeaders:")
		for _, h := range req.Event.Request.AdditionalHeaders {
			log.Printf("  %s: %v", h.Name, h.Value)
		}
	}

	// Get the partner ID from event.request.additionalHeaders
	partnerID := getHeaderValue(req.Event.Request.AdditionalHeaders, "x-b2b-usp-partner")
	if partnerID == "" {
		log.Printf("Warning: x-b2b-usp-partner header not found in AdditionalHeaders")
		resp := Response{
			ActionStatus: "SUCCESS",
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Partner ID from AdditionalHeaders: %s", partnerID)

	// Load entitlements.json
	entitlementsData, err := loadEntitlements()
	if err != nil {
		log.Printf("Error loading entitlements: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Find matching entitlements and create scopes
	var operations []OperationResponse
	for _, entitlement := range entitlementsData.Entitlements {
		if entitlement.Subject.Type == "partner" && entitlement.Subject.ID == partnerID {
			scope := fmt.Sprintf("%s:%s", entitlement.Subject.Type, entitlement.Action)
			operations = append(operations, OperationResponse{
				Op:   "add",
				Path: "/accessToken/scopes/-",
				Value: scope,
			})
			log.Printf("Added scope: %s for partner %s", scope, partnerID)
		}
	}

	// Return success response with actionStatus and operations
	resp := Response{
		ActionStatus: "SUCCESS",
		Operations:  operations,
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// getHeaderValue extracts the first value of a header from AdditionalHeaders array
func getHeaderValue(headers []Header, headerName string) string {
	for _, header := range headers {
		if header.Name == headerName && len(header.Value) > 0 {
			return header.Value[0] // Return the first value
		}
	}
	return ""
}

// loadEntitlements loads and parses the entitlements.json file
func loadEntitlements() (*EntitlementsData, error) {
	data, err := ioutil.ReadFile("entitlements.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read entitlements.json: %w", err)
	}

	var entitlementsData EntitlementsData
	if err := json.Unmarshal(data, &entitlementsData); err != nil {
		return nil, fmt.Errorf("failed to parse entitlements.json: %w", err)
	}

	return &entitlementsData, nil
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
