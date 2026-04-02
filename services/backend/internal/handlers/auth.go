package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	supabaseAuthorizePath = "/auth/v1/authorize"
	supabaseSignupPath    = "/auth/v1/signup"
	supabaseTokenPath     = "/auth/v1/token"
	supabaseUserPath      = "/auth/v1/user"
	supabaseLogoutPath    = "/auth/v1/logout"
)

type pendingAuthFlow struct {
	State     string
	Provider  string
	CreatedAt time.Time
}

type authSession struct {
	UserID               string
	Email                string
	Name                 string
	Provider             string
	AccessToken          string
	RefreshToken         string
	ProviderToken        string
	ProviderRefreshToken string
	TokenType            string
	ExpiresAt            time.Time
}

type AuthHandler struct {
	cfg    config.Config
	client *http.Client

	mu          sync.RWMutex
	pendingFlow *pendingAuthFlow
	session     *authSession
}

type supabaseRequestError struct {
	StatusCode int
	Message    string
}

func (e *supabaseRequestError) Error() string {
	return e.Message
}

type authStatusResponse struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Message  string `json:"message,omitempty"`
}

type googleFinalizeRequest struct {
	FlowState            string `json:"flow_state"`
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	ProviderToken        string `json:"provider_token"`
	ProviderRefreshToken string `json:"provider_refresh_token"`
	TokenType            string `json:"token_type"`
	ExpiresIn            int    `json:"expires_in"`
	Error                string `json:"error"`
	ErrorDescription     string `json:"error_description"`
}

type emailAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

type supabaseSessionResponse struct {
	AccessToken          string        `json:"access_token"`
	TokenType            string        `json:"token_type"`
	ExpiresIn            int           `json:"expires_in"`
	RefreshToken         string        `json:"refresh_token"`
	ProviderToken        string        `json:"provider_token"`
	ProviderRefreshToken string        `json:"provider_refresh_token"`
	User                 *supabaseUser `json:"user"`
	Error                string        `json:"error"`
	ErrorDescription     string        `json:"error_description"`
}

type supabaseUser struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	UserMetadata supabaseUserMetadata   `json:"user_metadata"`
	AppMetadata  supabaseAppMetadata    `json:"app_metadata"`
	Identities   []supabaseUserIdentity `json:"identities"`
}

type supabaseUserMetadata struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type supabaseAppMetadata struct {
	Provider  string   `json:"provider"`
	Providers []string `json:"providers"`
}

type supabaseUserIdentity struct {
	Provider string `json:"provider"`
}

func NewAuthHandler(cfg config.Config) *AuthHandler {
	return &AuthHandler{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (h *AuthHandler) GoogleStart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.isSupabaseConfigured() {
			writeJSON(w, http.StatusServiceUnavailable, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Supabase auth is not configured. Set SUPABASE_URL, SUPABASE_ANON_KEY, and SUPABASE_REDIRECT_URL.",
			})
			return
		}

		state, err := randomState()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Failed to start Google sign-in.",
			})
			return
		}

		h.mu.Lock()
		h.pendingFlow = &pendingAuthFlow{
			State:     state,
			Provider:  "google",
			CreatedAt: time.Now(),
		}
		h.mu.Unlock()

		query := url.Values{}
		query.Set("provider", "google")
		query.Set("redirect_to", withQueryParam(h.cfg.Supabase.RedirectURL, "flow_state", state))
		query.Set("access_type", "offline")
		query.Set("prompt", "consent")

		http.Redirect(w, r, h.cfg.Supabase.URL+supabaseAuthorizePath+"?"+query.Encode(), http.StatusFound)
	}
}

func (h *AuthHandler) GoogleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeOAuthHTML(w, "Complete Google sign-in", googleCallbackBodyHTML())
	}
}

func (h *AuthHandler) GoogleFinalize() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Method not allowed.",
			})
			return
		}

		var payload googleFinalizeRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Invalid Google auth payload.",
			})
			return
		}

		if payload.Error != "" {
			message := payload.ErrorDescription
			if message == "" {
				message = payload.Error
			}

			writeJSON(w, http.StatusBadRequest, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  message,
			})
			return
		}

		if !h.isValidFlowState(payload.FlowState, "google") {
			writeJSON(w, http.StatusBadRequest, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Google sign-in state was invalid or expired.",
			})
			return
		}

		if payload.AccessToken == "" {
			writeJSON(w, http.StatusBadRequest, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Google sign-in returned no Supabase access token.",
			})
			return
		}

		user, err := h.fetchSupabaseUser(payload.AccessToken)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

		session := newSessionFromSupabase("google", payload.AccessToken, payload.RefreshToken, payload.ProviderToken, payload.ProviderRefreshToken, payload.TokenType, payload.ExpiresIn, user)

		h.mu.Lock()
		h.session = session
		h.pendingFlow = nil
		h.mu.Unlock()

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "google",
			Status:   "connected",
			Email:    session.Email,
			Name:     session.Name,
		})
	}
}

func (h *AuthHandler) GoogleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := h.ensureSessionFresh()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

		if session == nil || session.Provider != "google" {
			writeJSON(w, http.StatusOK, authStatusResponse{
				Provider: "google",
				Status:   "disconnected",
			})
			return
		}

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "google",
			Status:   "connected",
			Email:    session.Email,
			Name:     session.Name,
		})
	}
}

func (h *AuthHandler) EmailConfirmed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeOAuthHTML(
			w,
			"Email confirmed",
			`<p>Your Kai account email is confirmed. Return to the Kai desktop app and sign in with your email and password.</p>
    <p class="muted">You can close this window once you are back in Kai.</p>`,
		)
	}
}

func (h *AuthHandler) AuthLanding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeOAuthHTML(
			w,
			"Kai authentication",
			rootAuthLandingBodyHTML(),
		)
	}
}

func (h *AuthHandler) EmailSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, ok := decodeEmailAuthRequest(w, r)
		if !ok {
			return
		}

		options := map[string]any{
			"email_redirect_to": h.cfg.Supabase.EmailRedirectURL,
		}
		if strings.TrimSpace(request.Name) != "" {
			options["data"] = map[string]any{"name": strings.TrimSpace(request.Name)}
		}

		body := map[string]any{
			"email":    request.Email,
			"password": request.Password,
			"options":  options,
		}

		var response supabaseSessionResponse
		if err := h.supabaseJSONRequest(http.MethodPost, supabaseSignupPath, nil, body, &response); err != nil {
			statusCode := http.StatusBadGateway
			if requestErr := new(supabaseRequestError); errors.As(err, &requestErr) {
				statusCode = requestErr.StatusCode
			}
			writeJSON(w, statusCode, authStatusResponse{
				Provider: "kai",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

		if response.AccessToken == "" || response.User == nil {
			writeJSON(w, http.StatusOK, authStatusResponse{
				Provider: "kai",
				Status:   "disconnected",
				Message:  "Account created. Check your email to finish confirmation.",
			})
			return
		}

		session := newSessionFromSupabase("email", response.AccessToken, response.RefreshToken, response.ProviderToken, response.ProviderRefreshToken, response.TokenType, response.ExpiresIn, response.User)

		h.mu.Lock()
		h.session = session
		h.mu.Unlock()

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "kai",
			Status:   "connected",
			Email:    session.Email,
			Name:     session.Name,
		})
	}
}

func (h *AuthHandler) EmailSignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, ok := decodeEmailAuthRequest(w, r)
		if !ok {
			return
		}

		query := url.Values{}
		query.Set("grant_type", "password")

		body := map[string]any{
			"email":    request.Email,
			"password": request.Password,
		}

		var response supabaseSessionResponse
		if err := h.supabaseJSONRequest(http.MethodPost, supabaseTokenPath, query, body, &response); err != nil {
			statusCode := http.StatusUnauthorized
			if requestErr := new(supabaseRequestError); errors.As(err, &requestErr) {
				statusCode = requestErr.StatusCode
			}
			writeJSON(w, statusCode, authStatusResponse{
				Provider: "kai",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

		if response.AccessToken == "" || response.User == nil {
			writeJSON(w, http.StatusUnauthorized, authStatusResponse{
				Provider: "kai",
				Status:   "error",
				Message:  "Supabase sign-in returned no session.",
			})
			return
		}

		session := newSessionFromSupabase("email", response.AccessToken, response.RefreshToken, response.ProviderToken, response.ProviderRefreshToken, response.TokenType, response.ExpiresIn, response.User)

		h.mu.Lock()
		h.session = session
		h.mu.Unlock()

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "kai",
			Status:   "connected",
			Email:    session.Email,
			Name:     session.Name,
		})
	}
}

func (h *AuthHandler) Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := h.ensureSessionFresh()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{
				Provider: "kai",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

		if session == nil {
			writeJSON(w, http.StatusOK, authStatusResponse{
				Provider: "kai",
				Status:   "disconnected",
			})
			return
		}

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: session.Provider,
			Status:   "connected",
			Email:    session.Email,
			Name:     session.Name,
		})
	}
}

func (h *AuthHandler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		session := h.session
		h.session = nil
		h.pendingFlow = nil
		h.mu.Unlock()

		if session != nil && session.AccessToken != "" {
			_ = h.supabaseAuthorizedRequest(http.MethodPost, supabaseLogoutPath, session.AccessToken, nil)
		}

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "kai",
			Status:   "disconnected",
		})
	}
}

func decodeEmailAuthRequest(w http.ResponseWriter, r *http.Request) (emailAuthRequest, bool) {
	var request emailAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, authStatusResponse{
			Provider: "kai",
			Status:   "error",
			Message:  "Invalid auth request body.",
		})
		return request, false
	}

	request.Email = strings.TrimSpace(request.Email)
	request.Password = strings.TrimSpace(request.Password)

	if request.Email == "" || request.Password == "" {
		writeJSON(w, http.StatusBadRequest, authStatusResponse{
			Provider: "kai",
			Status:   "error",
			Message:  "Email and password are required.",
		})
		return request, false
	}

	return request, true
}

func (h *AuthHandler) isSupabaseConfigured() bool {
	return h.cfg.Supabase.URL != "" && h.cfg.Supabase.AnonKey != "" && h.cfg.Supabase.RedirectURL != ""
}

func (h *AuthHandler) isValidFlowState(state string, provider string) bool {
	if state == "" {
		return false
	}

	h.mu.RLock()
	flow := h.pendingFlow
	h.mu.RUnlock()

	if flow == nil {
		return false
	}

	if flow.Provider != provider || flow.State != state {
		return false
	}

	return time.Since(flow.CreatedAt) <= 10*time.Minute
}

func (h *AuthHandler) ensureSessionFresh() (*authSession, error) {
	h.mu.RLock()
	session := h.session
	h.mu.RUnlock()

	if session == nil {
		return nil, nil
	}

	if session.ExpiresAt.IsZero() || time.Until(session.ExpiresAt) > time.Minute {
		return session, nil
	}

	if session.RefreshToken == "" {
		return session, nil
	}

	refreshed, err := h.refreshSupabaseSession(session.RefreshToken)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	h.session = refreshed
	updated := h.session
	h.mu.Unlock()

	return updated, nil
}

func (h *AuthHandler) refreshSupabaseSession(refreshToken string) (*authSession, error) {
	query := url.Values{}
	query.Set("grant_type", "refresh_token")

	body := map[string]any{
		"refresh_token": refreshToken,
	}

	var response supabaseSessionResponse
	if err := h.supabaseJSONRequest(http.MethodPost, supabaseTokenPath, query, body, &response); err != nil {
		return nil, err
	}

	if response.AccessToken == "" {
		return nil, fmt.Errorf("supabase refresh returned no access token")
	}

	user := response.User
	if user == nil {
		fetchedUser, err := h.fetchSupabaseUser(response.AccessToken)
		if err != nil {
			return nil, err
		}
		user = fetchedUser
	}

	return newSessionFromSupabase(currentProvider(user), response.AccessToken, firstNonEmpty(response.RefreshToken, refreshToken), response.ProviderToken, response.ProviderRefreshToken, response.TokenType, response.ExpiresIn, user), nil
}

func (h *AuthHandler) fetchSupabaseUser(accessToken string) (*supabaseUser, error) {
	request, err := http.NewRequest(http.MethodGet, h.cfg.Supabase.URL+supabaseUserPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase user request: %w", err)
	}

	request.Header.Set("apikey", h.cfg.Supabase.AnonKey)
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := h.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Supabase user: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Supabase user response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("supabase user lookup failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var user supabaseUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to decode Supabase user response: %w", err)
	}

	if user.Email == "" {
		return nil, fmt.Errorf("supabase user response did not include an email")
	}

	return &user, nil
}

func (h *AuthHandler) supabaseJSONRequest(method string, path string, query url.Values, body any, target any) error {
	if !h.isSupabaseConfigured() {
		return fmt.Errorf("supabase auth is not configured")
	}

	endpoint := h.cfg.Supabase.URL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var payload io.Reader
	if body != nil {
		buffer, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
		payload = strings.NewReader(string(buffer))
	}

	request, err := http.NewRequest(method, endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to create Supabase request: %w", err)
	}

	request.Header.Set("apikey", h.cfg.Supabase.AnonKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := h.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to reach Supabase: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read Supabase response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = "unknown Supabase auth error"
		}
		return &supabaseRequestError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("supabase auth failed with HTTP %d: %s", response.StatusCode, message),
		}
	}

	if target == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("failed to decode Supabase response: %w", err)
	}

	return nil
}

func (h *AuthHandler) supabaseAuthorizedRequest(method string, path string, accessToken string, body any) error {
	endpoint := h.cfg.Supabase.URL + path

	var payload io.Reader
	if body != nil {
		buffer, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = strings.NewReader(string(buffer))
	}

	request, err := http.NewRequest(method, endpoint, payload)
	if err != nil {
		return err
	}

	request.Header.Set("apikey", h.cfg.Supabase.AnonKey)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := h.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("supabase authorized request failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func newSessionFromSupabase(provider string, accessToken string, refreshToken string, providerToken string, providerRefreshToken string, tokenType string, expiresIn int, user *supabaseUser) *authSession {
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	if expiresIn <= 0 {
		expiresAt = time.Time{}
	}

	return &authSession{
		UserID:               user.ID,
		Email:                user.Email,
		Name:                 pickDisplayName(user),
		Provider:             provider,
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		ProviderToken:        providerToken,
		ProviderRefreshToken: providerRefreshToken,
		TokenType:            tokenType,
		ExpiresAt:            expiresAt,
	}
}

func pickDisplayName(user *supabaseUser) string {
	switch {
	case strings.TrimSpace(user.UserMetadata.Name) != "":
		return strings.TrimSpace(user.UserMetadata.Name)
	case strings.TrimSpace(user.UserMetadata.FullName) != "":
		return strings.TrimSpace(user.UserMetadata.FullName)
	default:
		return ""
	}
}

func currentProvider(user *supabaseUser) string {
	if strings.TrimSpace(user.AppMetadata.Provider) != "" {
		return strings.TrimSpace(user.AppMetadata.Provider)
	}

	for _, identity := range user.Identities {
		if strings.TrimSpace(identity.Provider) != "" {
			return strings.TrimSpace(identity.Provider)
		}
	}

	return "email"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func withQueryParam(raw string, key string, value string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
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

func writeOAuthHTML(w http.ResponseWriter, title string, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <title>%s</title>
    <style>
      body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; background: #111827; color: #f3f4f6; display: grid; place-items: center; min-height: 100vh; margin: 0; }
      main { width: min(560px, calc(100vw - 32px)); padding: 24px; border-radius: 18px; background: rgba(255,255,255,0.06); box-shadow: 0 20px 60px rgba(0,0,0,0.25); }
      h1 { margin: 0 0 12px; font-size: 1.4rem; }
      p { margin: 0; color: rgba(243,244,246,0.78); line-height: 1.5; }
      .muted { margin-top: 12px; font-size: 0.95rem; color: rgba(243,244,246,0.64); }
      .success { color: #bfdbfe; }
      .error { color: #fecaca; }
    </style>
  </head>
  <body>
    <main>
      <h1>%s</h1>
      %s
    </main>
  </body>
</html>`, title, title, body)))
}

func googleCallbackBodyHTML() string {
	return `<p id="status">Completing Google sign-in for Kai...</p>
    <p class="muted" id="detail">You can close this window after Kai confirms the connection.</p>
    <script>
      const statusNode = document.getElementById("status");
      const detailNode = document.getElementById("detail");
      const url = new URL(window.location.href);
      const hashParams = new URLSearchParams(window.location.hash.startsWith("#") ? window.location.hash.slice(1) : "");
      const searchParams = new URLSearchParams(url.search);

      const payload = {
        flow_state: searchParams.get("flow_state") || "",
        access_token: hashParams.get("access_token") || searchParams.get("access_token") || "",
        refresh_token: hashParams.get("refresh_token") || searchParams.get("refresh_token") || "",
        provider_token: hashParams.get("provider_token") || searchParams.get("provider_token") || "",
        provider_refresh_token: hashParams.get("provider_refresh_token") || searchParams.get("provider_refresh_token") || "",
        token_type: hashParams.get("token_type") || searchParams.get("token_type") || "",
        expires_in: Number(hashParams.get("expires_in") || searchParams.get("expires_in") || "0"),
        error: hashParams.get("error") || searchParams.get("error") || "",
        error_description: hashParams.get("error_description") || searchParams.get("error_description") || ""
      };

      if (payload.error) {
        statusNode.textContent = "Google sign-in failed";
        statusNode.className = "error";
        detailNode.textContent = payload.error_description || payload.error;
      } else if (!payload.access_token) {
        statusNode.textContent = "Google sign-in failed";
        statusNode.className = "error";
        detailNode.textContent = "Supabase did not return an access token to Kai.";
      } else {
        fetch("/auth/google/session", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        })
          .then(async (response) => {
            const data = await response.json().catch(() => ({}));
            if (!response.ok || data.status === "error") {
              throw new Error(data.message || ("Google sign-in failed with HTTP " + response.status));
            }

            statusNode.textContent = "Google connected";
            statusNode.className = "success";
            detailNode.textContent = data.email
              ? "Kai is now connected to " + data.email + ". You can close this window."
              : "Kai is now connected. You can close this window.";
          })
          .catch((error) => {
            statusNode.textContent = "Google sign-in failed";
            statusNode.className = "error";
            detailNode.textContent = error instanceof Error ? error.message : String(error);
          });
      }
    </script>`
}

func rootAuthLandingBodyHTML() string {
	return `<p id="status">Checking your Kai authentication result...</p>
    <p class="muted" id="detail">You can close this window and return to Kai once this completes.</p>
    <script>
      const statusNode = document.getElementById("status");
      const detailNode = document.getElementById("detail");
      const hashParams = new URLSearchParams(window.location.hash.startsWith("#") ? window.location.hash.slice(1) : "");
      const accessToken = hashParams.get("access_token") || "";
      const authType = hashParams.get("type") || "";
      const error = hashParams.get("error_description") || hashParams.get("error") || "";

      if (error) {
        statusNode.textContent = "Authentication failed";
        statusNode.className = "error";
        detailNode.textContent = error;
      } else if (authType === "signup" && accessToken) {
        statusNode.textContent = "Email confirmed";
        statusNode.className = "success";
        detailNode.textContent = "Your Kai account is verified. Return to the Kai desktop app and sign in with your email and password.";
      } else if (accessToken) {
        statusNode.textContent = "Authentication complete";
        statusNode.className = "success";
        detailNode.textContent = "Kai received an auth response. Return to the desktop app to continue.";
      } else {
        statusNode.textContent = "Nothing to complete";
        statusNode.className = "error";
        detailNode.textContent = "Kai did not receive an authentication token on this page.";
      }

      if (window.location.hash) {
        history.replaceState(null, "", window.location.pathname);
      }
    </script>`
}
