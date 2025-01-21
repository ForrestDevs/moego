package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo
	},
}

// TokenCache manages ephemeral tokens with expiration
type TokenCache struct {
	sync.RWMutex
	token     *SessionResponse
	expiresAt time.Time
}

var tokenCache = &TokenCache{}

func (tc *TokenCache) get() *SessionResponse {
	tc.RLock()
	defer tc.RUnlock()

	if tc.token != nil && time.Now().Before(tc.expiresAt) {
		return tc.token
	}
	return nil
}

func (tc *TokenCache) set(token *SessionResponse) {
	tc.Lock()
	defer tc.Unlock()

	tc.token = token
	// Set expiration to 50 seconds (tokens expire after 60 seconds)
	tc.expiresAt = time.Now().Add(50 * time.Second)
}

// Logger middleware
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

type SessionResponse struct {
	ClientSecret struct {
		Value string `json:"value"`
	} `json:"client_secret"`
}

type WebRTCMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func createEphemeralToken() (*SessionResponse, error) {
	// Check cache first
	if token := tokenCache.get(); token != nil {
		log.Printf("Using cached ephemeral token")
		return token, nil
	}

	log.Printf("Creating new ephemeral token")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	payload := map[string]interface{}{
		"model": "gpt-4o-mini-realtime-preview-2024-12-17",
		"voice": "verse",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/realtime/sessions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var sessionResp SessionResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return nil, err
	}

	// Cache the token
	tokenCache.set(&sessionResp)
	return &sessionResp, nil
}

// Helper function to send error messages to the WebSocket client
func sendError(conn *websocket.Conn, message string) {
	if err := conn.WriteJSON(map[string]string{
		"type":    "error",
		"message": message,
	}); err != nil {
		log.Printf("Error sending error message to client: %v", err)
	}
}

func handleWebRTCSignaling(w http.ResponseWriter, r *http.Request) {
	log.Printf("New token request from %s", r.RemoteAddr)

	// Create ephemeral token
	session, err := createEphemeralToken()
	if err != nil {
		log.Printf("Failed to create ephemeral token: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return token to client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": session.ClientSecret.Value,
	})
}

func main() {
	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Starting server...")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory:", err)
	}
	log.Printf("Current working directory: %s", cwd)

	// Verify static directory exists
	staticDir := filepath.Join(cwd, "examples", "rtc", "static")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Fatal("Static directory not found:", staticDir)
	}
	log.Printf("Static directory found: %s", staticDir)

	// Create a new ServeMux
	mux := http.NewServeMux()

	// WebSocket endpoint for WebRTC signaling
	mux.HandleFunc("/rtc", handleWebRTCSignaling)

	// Serve static files with logging
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", loggerMiddleware(fs))

	// Verify index.html exists
	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		log.Fatal("index.html not found:", indexPath)
	}
	log.Printf("index.html found: %s", indexPath)

	// Start the server
	addr := ":8080"
	log.Printf("Server starting on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
