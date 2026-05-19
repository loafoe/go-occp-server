package ocpp

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

// Authenticator handles charge point authentication
type Authenticator struct {
	credentials map[string]string
	mu          sync.RWMutex
	enabled     bool
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		credentials: make(map[string]string),
		enabled:     false,
	}
}

// LoadFromFile loads credentials from a JSON file
// Format: {"chargePointId": "password", ...}
func (a *Authenticator) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var creds map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		return err
	}

	a.mu.Lock()
	a.credentials = creds
	a.enabled = true
	a.mu.Unlock()

	log.Printf("Loaded %d charge point credentials", len(creds))
	return nil
}

// LoadFromEnv loads credentials from environment variable
// Format: JSON string {"chargePointId": "password", ...}
func (a *Authenticator) LoadFromEnv(envVar string) error {
	data := os.Getenv(envVar)
	if data == "" {
		return nil
	}

	var creds map[string]string
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return err
	}

	a.mu.Lock()
	a.credentials = creds
	a.enabled = true
	a.mu.Unlock()

	log.Printf("Loaded %d charge point credentials from env", len(creds))
	return nil
}

// SetCredentials sets credentials directly
func (a *Authenticator) SetCredentials(creds map[string]string) {
	a.mu.Lock()
	a.credentials = creds
	a.enabled = len(creds) > 0
	a.mu.Unlock()
}

// Enable enables or disables authentication
func (a *Authenticator) Enable(enabled bool) {
	a.mu.Lock()
	a.enabled = enabled
	a.mu.Unlock()
}

// IsEnabled returns whether authentication is enabled
func (a *Authenticator) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// Authenticate validates charge point credentials from Basic Auth header
// Returns the charge point ID if valid, empty string if invalid
func (a *Authenticator) Authenticate(chargePointID, authHeader string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.enabled {
		return true
	}

	expectedPassword, exists := a.credentials[chargePointID]
	if !exists {
		log.Printf("Auth failed: unknown charge point ID: %s", chargePointID)
		return false
	}

	if authHeader == "" {
		log.Printf("Auth failed: no Authorization header for %s", chargePointID)
		return false
	}

	// Parse Basic Auth header
	username, password, ok := parseBasicAuth(authHeader)
	if !ok {
		log.Printf("Auth failed: invalid Authorization header for %s", chargePointID)
		return false
	}

	// Username should match charge point ID
	if username != chargePointID {
		log.Printf("Auth failed: username mismatch for %s (got %s)", chargePointID, username)
		return false
	}

	if password != expectedPassword {
		log.Printf("Auth failed: invalid password for %s", chargePointID)
		return false
	}

	return true
}

// parseBasicAuth parses an HTTP Basic Authentication header
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}

	username, password, found := strings.Cut(string(decoded), ":")
	if !found {
		return "", "", false
	}

	return username, password, true
}
