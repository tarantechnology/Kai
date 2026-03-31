package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/taran/kai/services/backend/internal/config"
)

const (
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
	googleUserURL  = "https://openidconnect.googleapis.com/v1/userinfo"
)

type googleAuthState struct {
	Value     string
	CreatedAt time.Time
}

type googleConnection struct {
	Email        string
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	Expiry       time.Time
}

type AuthHandler struct {
	cfg config.Config

	mu              sync.RWMutex
	currentState    *googleAuthState
	googleConnected *googleConnection
}

type authStatusResponse struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	Email    string `json:"email,omitempty"`
	Message  string `json:"message,omitempty"`
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
}

type googleUserInfoResponse struct {
	Email string `json:"email"`
}

func NewAuthHandler(cfg config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

func (h *AuthHandler) GoogleStart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.cfg.Google.ClientID == "" || h.cfg.Google.ClientSecret == "" || h.cfg.Google.RedirectURL == "" {
			writeJSON(w, http.StatusServiceUnavailable, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Google OAuth is not configured. Set GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL.",
			})
			return
		}

		state, err := randomState()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Failed to generate OAuth state.",
			})
			return
		}

		h.mu.Lock()
		h.currentState = &googleAuthState{
			Value:     state,
			CreatedAt: time.Now(),
		}
		h.mu.Unlock()

		query := url.Values{}
		query.Set("client_id", h.cfg.Google.ClientID)
		query.Set("redirect_uri", h.cfg.Google.RedirectURL)
		query.Set("response_type", "code")
		query.Set("scope", strings.Join(h.cfg.Google.Scopes, " "))
		query.Set("state", state)
		query.Set("access_type", "offline")
		query.Set("prompt", "consent")

		http.Redirect(w, r, googleAuthURL+"?"+query.Encode(), http.StatusFound)
	}
}

func (h *AuthHandler) GoogleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			writeOAuthHTML(w, "Google sign-in failed", "Google did not return the required OAuth code/state.")
			return
		}

		if !h.isValidState(state) {
			writeOAuthHTML(w, "Google sign-in failed", "The OAuth state was invalid or expired.")
			return
		}

		token, err := h.exchangeGoogleCode(code)
		if err != nil {
			writeOAuthHTML(w, "Google sign-in failed", err.Error())
			return
		}

		email, err := fetchGoogleEmail(token.AccessToken)
		if err != nil {
			writeOAuthHTML(w, "Google sign-in partial", "Tokens were created, but Kai could not fetch your Google profile email.")
			return
		}

		h.mu.Lock()
		h.googleConnected = &googleConnection{
			Email:        email,
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    token.TokenType,
			Scope:        token.Scope,
			Expiry:       time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
		}
		h.currentState = nil
		h.mu.Unlock()

		writeOAuthHTML(w, "Google connected", fmt.Sprintf("Kai is now connected to %s. You can close this window.", email))
	}
}

func (h *AuthHandler) GoogleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		connection := h.googleConnected
		h.mu.RUnlock()

		if connection == nil {
			writeJSON(w, http.StatusOK, authStatusResponse{
				Provider: "google",
				Status:   "disconnected",
			})
			return
		}

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "google",
			Status:   "connected",
			Email:    connection.Email,
		})
	}
}

func (h *AuthHandler) isValidState(state string) bool {
	h.mu.RLock()
	current := h.currentState
	h.mu.RUnlock()

	if current == nil {
		return false
	}

	if time.Since(current.CreatedAt) > 10*time.Minute {
		return false
	}

	return current.Value == state
}

func (h *AuthHandler) exchangeGoogleCode(code string) (*googleTokenResponse, error) {
	values := url.Values{}
	values.Set("client_id", h.cfg.Google.ClientID)
	values.Set("client_secret", h.cfg.Google.ClientSecret)
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", h.cfg.Google.RedirectURL)

	response, err := http.PostForm(googleTokenURL, values)
	if err != nil {
		return nil, fmt.Errorf("Failed to exchange Google OAuth code: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read Google token response: %w", err)
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("Google token exchange failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed googleTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("Failed to decode Google token response: %w", err)
	}

	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("Google token exchange returned no access token")
	}

	return &parsed, nil
}

func fetchGoogleEmail(accessToken string) (string, error) {
	request, err := http.NewRequest(http.MethodGet, googleUserURL, nil)
	if err != nil {
		return "", err
	}

	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode >= 400 {
		return "", fmt.Errorf("Google userinfo request failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed googleUserInfoResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}

	if parsed.Email == "" {
		return "", fmt.Errorf("Google userinfo response did not include an email")
	}

	return parsed.Email, nil
}

func randomState() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeOAuthHTML(w http.ResponseWriter, title string, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <title>%s</title>
    <style>
      body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; background: #111827; color: #f3f4f6; display: grid; place-items: center; min-height: 100vh; margin: 0; }
      main { width: min(520px, calc(100vw - 32px)); padding: 24px; border-radius: 18px; background: rgba(255,255,255,0.06); box-shadow: 0 20px 60px rgba(0,0,0,0.25); }
      h1 { margin: 0 0 12px; font-size: 1.4rem; }
      p { margin: 0; color: rgba(243,244,246,0.78); line-height: 1.5; }
    </style>
  </head>
  <body>
    <main>
      <h1>%s</h1>
      <p>%s</p>
    </main>
  </body>
</html>`, title, title, message)))
}
