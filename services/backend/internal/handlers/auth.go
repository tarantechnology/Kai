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
	supabaseRESTPath      = "/rest/v1"

	googleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	googleUserInfoURL  = "https://openidconnect.googleapis.com/v1/userinfo"
)

type pendingAuthFlow struct {
	State     string
	Provider  string
	Intent    string
	CreatedAt time.Time
}

type authSession struct {
	UserID                 string
	Email                  string
	Name                   string
	AuthProvider           string
	AccessToken            string
	RefreshToken           string
	TokenType              string
	ExpiresAt              time.Time
	GoogleConnected        bool
	GoogleEmail            string
	GoogleName             string
	GoogleProviderToken    string
	GoogleRefreshToken     string
	GoogleTokenType        string
	GoogleScopes           string
	GoogleExpiresAt        time.Time
	GoogleProviderIdentity string
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
	Provider             string `json:"provider"`
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

type connectedPlatformRecord struct {
	UserID        string `json:"user_id"`
	Provider      string `json:"provider"`
	ExternalEmail string `json:"external_email"`
	ExternalName  string `json:"external_name"`
	Status        string `json:"status"`
}

type providerTokenRecord struct {
	UserID       string     `json:"user_id"`
	Provider     string     `json:"provider"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	Scopes       string     `json:"scopes"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

type googleUserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
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
			Intent:    normalizeGoogleIntent(r.URL.Query().Get("intent")),
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

func (h *AuthHandler) GoogleConnectStart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.isGoogleOAuthConfigured() {
			writeJSON(w, http.StatusServiceUnavailable, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Google OAuth is not configured for provider connection.",
			})
			return
		}

		h.mu.RLock()
		session := h.session
		h.mu.RUnlock()

		if session == nil {
			writeJSON(w, http.StatusUnauthorized, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Sign in to Kai before connecting Google.",
			})
			return
		}

		state, err := randomState()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Failed to start Google connection.",
			})
			return
		}

		h.mu.Lock()
		h.pendingFlow = &pendingAuthFlow{
			State:     state,
			Provider:  "google",
			Intent:    "connect",
			CreatedAt: time.Now(),
		}
		h.mu.Unlock()

		query := url.Values{}
		query.Set("client_id", h.cfg.Google.ClientID)
		query.Set("redirect_uri", h.cfg.Google.ConnectRedirectURL)
		query.Set("response_type", "code")
		query.Set("access_type", "offline")
		query.Set("prompt", "consent")
		query.Set("include_granted_scopes", "true")
		query.Set("state", state)
		query.Set("scope", strings.Join(h.cfg.Google.Scopes, " "))

		http.Redirect(w, r, googleAuthorizeURL+"?"+query.Encode(), http.StatusFound)
	}
}

func (h *AuthHandler) GoogleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeOAuthHTML(w, "Complete Google sign-in", googleCallbackBodyHTML())
	}
}

func (h *AuthHandler) GoogleConnectCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if errValue := strings.TrimSpace(r.URL.Query().Get("error")); errValue != "" {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", errValue))
			return
		}

		flow := h.lookupPendingFlow(state, "google")
		if flow == nil || flow.Intent != "connect" {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", "Google connection state was invalid or expired."))
			return
		}

		h.mu.RLock()
		session := h.session
		h.mu.RUnlock()
		if session == nil {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", "Sign in to Kai before connecting Google."))
			return
		}

		token, err := h.exchangeGoogleCode(code)
		if err != nil {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", err.Error()))
			return
		}

		userInfo, err := h.fetchGoogleUserInfo(token.AccessToken)
		if err != nil {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", err.Error()))
			return
		}

		h.mu.Lock()
		if h.session != nil {
			h.session.GoogleConnected = true
			h.session.GoogleEmail = firstNonEmpty(userInfo.Email, h.session.Email)
			h.session.GoogleName = firstNonEmpty(userInfo.Name, h.session.Name)
			h.session.GoogleProviderToken = token.AccessToken
			h.session.GoogleRefreshToken = firstNonEmpty(token.RefreshToken, h.session.GoogleRefreshToken)
			h.session.GoogleTokenType = token.TokenType
			h.session.GoogleScopes = strings.Join(h.cfg.Google.Scopes, ",")
			h.session.GoogleExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
			h.session.GoogleProviderIdentity = "google"
		}
		h.pendingFlow = nil
		currentSession := h.session
		h.mu.Unlock()

		if currentSession == nil {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", "Kai session expired while connecting Google."))
			return
		}

		if err := h.upsertConnectedPlatform(
			currentSession.UserID,
			"google",
			firstNonEmpty(userInfo.Email, currentSession.Email),
			firstNonEmpty(userInfo.Name, currentSession.Name),
		); err != nil {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", err.Error()))
			return
		}

		if err := h.upsertProviderToken(currentSession.UserID, "google", token.AccessToken, firstNonEmpty(token.RefreshToken, currentSession.GoogleRefreshToken), token.TokenType, strings.Join(h.cfg.Google.Scopes, ","), time.Now().Add(time.Duration(token.ExpiresIn)*time.Second)); err != nil {
			writeOAuthHTML(w, "Google connection failed", oauthRedirectBodyHTML("google", "connect", "error", err.Error()))
			return
		}

		writeOAuthHTML(w, "Google connected", oauthRedirectBodyHTML("google", "connect", "success", "Google Calendar is now connected to Kai."))
	}
}

func (h *AuthHandler) GoogleFinalize() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.finalizeAuthSession(w, r, "google")
	}
}

func (h *AuthHandler) FinalizeSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.finalizeAuthSession(w, r, "kai")
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

		if session == nil || !session.GoogleConnected {
			writeJSON(w, http.StatusOK, authStatusResponse{
				Provider: "google",
				Status:   "disconnected",
			})
			return
		}

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "google",
			Status:   "connected",
			Email:    firstNonEmpty(session.GoogleEmail, session.Email),
			Name:     firstNonEmpty(session.GoogleName, session.Name),
		})
	}
}

func (h *AuthHandler) GoogleDisconnect() http.HandlerFunc {
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

		if session == nil || !session.GoogleConnected {
			writeJSON(w, http.StatusOK, authStatusResponse{
				Provider: "google",
				Status:   "disconnected",
			})
			return
		}

		h.mu.Lock()
		if h.session != nil {
			h.session.GoogleConnected = false
			h.session.GoogleEmail = ""
			h.session.GoogleName = ""
			h.session.GoogleProviderToken = ""
			h.session.GoogleRefreshToken = ""
			h.session.GoogleTokenType = ""
			h.session.GoogleScopes = ""
			h.session.GoogleExpiresAt = time.Time{}
			h.session.GoogleProviderIdentity = ""
		}
		h.mu.Unlock()

		if session.UserID != "" {
			if err := h.deleteConnectedPlatform(session.UserID, "google"); err != nil {
				writeJSON(w, http.StatusBadGateway, authStatusResponse{
					Provider: "google",
					Status:   "error",
					Message:  err.Error(),
				})
				return
			}
			if err := h.deleteProviderToken(session.UserID, "google"); err != nil {
				writeJSON(w, http.StatusBadGateway, authStatusResponse{
					Provider: "google",
					Status:   "error",
					Message:  err.Error(),
				})
				return
			}
		}

		writeJSON(w, http.StatusOK, authStatusResponse{
			Provider: "google",
			Status:   "disconnected",
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

		session := newSessionFromSupabase("email", response.AccessToken, response.RefreshToken, response.TokenType, response.ExpiresIn, response.User)
		if err := h.hydrateConnectedPlatforms(session); err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{
				Provider: "kai",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

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

func (h *AuthHandler) finalizeAuthSession(w http.ResponseWriter, r *http.Request, fallbackProvider string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, authStatusResponse{
			Provider: fallbackProvider,
			Status:   "error",
			Message:  "Method not allowed.",
		})
		return
	}

	var payload googleFinalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, authStatusResponse{
			Provider: fallbackProvider,
			Status:   "error",
			Message:  "Invalid auth payload.",
		})
		return
	}

	if payload.Error != "" {
		message := payload.ErrorDescription
		if message == "" {
			message = payload.Error
		}

		writeJSON(w, http.StatusBadRequest, authStatusResponse{
			Provider: fallbackProvider,
			Status:   "error",
			Message:  message,
		})
		return
	}

	if payload.AccessToken == "" {
		writeJSON(w, http.StatusBadRequest, authStatusResponse{
			Provider: fallbackProvider,
			Status:   "error",
			Message:  "Supabase returned no access token to Kai.",
		})
		return
	}

	var flow *pendingAuthFlow
	if payload.FlowState != "" {
		flow = h.lookupPendingFlow(payload.FlowState, "google")
		if flow == nil {
			writeJSON(w, http.StatusBadRequest, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Google sign-in state was invalid or expired.",
			})
			return
		}
	}

	user, err := h.fetchSupabaseUser(payload.AccessToken)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, authStatusResponse{
			Provider: fallbackProvider,
			Status:   "error",
			Message:  err.Error(),
		})
		return
	}

	intent := "login"
	if flow != nil && flow.Intent != "" {
		intent = flow.Intent
	}

	var session *authSession

	if intent == "connect" {
		h.mu.RLock()
		currentSession := h.session
		h.mu.RUnlock()
		if currentSession == nil {
			writeJSON(w, http.StatusUnauthorized, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  "Sign in to Kai before connecting Google.",
			})
			return
		}

		nextSession := *currentSession
		nextSession.GoogleConnected = true
		nextSession.GoogleEmail = user.Email
		nextSession.GoogleName = pickDisplayName(user)
		nextSession.GoogleProviderToken = payload.ProviderToken
		nextSession.GoogleRefreshToken = payload.ProviderRefreshToken
		nextSession.GoogleProviderIdentity = currentProvider(user)
		if err := h.upsertConnectedPlatform(
			nextSession.UserID,
			"google",
			firstNonEmpty(user.Email, nextSession.Email),
			firstNonEmpty(pickDisplayName(user), nextSession.Name),
		); err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{
				Provider: "google",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

		h.mu.Lock()
		h.session = &nextSession
		if payload.FlowState != "" {
			h.pendingFlow = nil
		}
		session = h.session
		h.mu.Unlock()
	} else {
		authProvider := strings.TrimSpace(payload.Provider)
		if authProvider == "" {
			authProvider = currentProvider(user)
		}
		if authProvider == "" {
			authProvider = fallbackProvider
		}

		session = newSessionFromSupabase(
			authProvider,
			payload.AccessToken,
			payload.RefreshToken,
			payload.TokenType,
			payload.ExpiresIn,
			user,
		)
		if authProvider == "google" || payload.ProviderToken != "" {
			session.GoogleConnected = true
			session.GoogleEmail = user.Email
			session.GoogleName = pickDisplayName(user)
			session.GoogleProviderToken = payload.ProviderToken
			session.GoogleRefreshToken = payload.ProviderRefreshToken
			session.GoogleProviderIdentity = currentProvider(user)
		}
		if err := h.hydrateConnectedPlatforms(session); err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{
				Provider: authProvider,
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}
		if session.GoogleConnected {
			if err := h.upsertConnectedPlatform(
				session.UserID,
				"google",
				firstNonEmpty(session.GoogleEmail, session.Email),
				firstNonEmpty(session.GoogleName, session.Name),
			); err != nil {
				writeJSON(w, http.StatusBadGateway, authStatusResponse{
					Provider: authProvider,
					Status:   "error",
					Message:  err.Error(),
				})
				return
			}
		}
		h.mu.Lock()
		h.session = session
		if payload.FlowState != "" {
			h.pendingFlow = nil
		}
		h.mu.Unlock()
	}

	writeJSON(w, http.StatusOK, authStatusResponse{
		Provider: session.AuthProvider,
		Status:   "connected",
		Email:    session.Email,
		Name:     session.Name,
	})
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

		session := newSessionFromSupabase("email", response.AccessToken, response.RefreshToken, response.TokenType, response.ExpiresIn, response.User)
		if err := h.hydrateConnectedPlatforms(session); err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{
				Provider: "kai",
				Status:   "error",
				Message:  err.Error(),
			})
			return
		}

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
			Provider: session.AuthProvider,
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

func (h *AuthHandler) isGoogleOAuthConfigured() bool {
	return h.cfg.Google.ClientID != "" && h.cfg.Google.ClientSecret != "" && h.cfg.Google.ConnectRedirectURL != ""
}

func (h *AuthHandler) lookupPendingFlow(state string, provider string) *pendingAuthFlow {
	if state == "" {
		return nil
	}

	h.mu.RLock()
	flow := h.pendingFlow
	h.mu.RUnlock()

	if flow == nil {
		return nil
	}

	if flow.Provider != provider || flow.State != state {
		return nil
	}

	if time.Since(flow.CreatedAt) > 10*time.Minute {
		return nil
	}

	return flow
}

func (h *AuthHandler) ensureSessionFresh() (*authSession, error) {
	h.mu.RLock()
	session := h.session
	h.mu.RUnlock()

	if session == nil {
		return nil, nil
	}

	if session.ExpiresAt.IsZero() || time.Until(session.ExpiresAt) > time.Minute {
		return h.ensureGoogleProviderFresh(session)
	}

	if session.RefreshToken == "" {
		return session, nil
	}

	refreshed, err := h.refreshSupabaseSession(session)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	h.session = refreshed
	updated := h.session
	h.mu.Unlock()

	return h.ensureGoogleProviderFresh(updated)
}

func (h *AuthHandler) refreshSupabaseSession(session *authSession) (*authSession, error) {
	query := url.Values{}
	query.Set("grant_type", "refresh_token")

	body := map[string]any{
		"refresh_token": session.RefreshToken,
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

	refreshed := newSessionFromSupabase(session.AuthProvider, response.AccessToken, firstNonEmpty(response.RefreshToken, session.RefreshToken), response.TokenType, response.ExpiresIn, user)
	refreshed.GoogleConnected = session.GoogleConnected
	refreshed.GoogleEmail = session.GoogleEmail
	refreshed.GoogleName = session.GoogleName
	refreshed.GoogleProviderToken = session.GoogleProviderToken
	refreshed.GoogleRefreshToken = session.GoogleRefreshToken
	refreshed.GoogleTokenType = session.GoogleTokenType
	refreshed.GoogleScopes = session.GoogleScopes
	refreshed.GoogleExpiresAt = session.GoogleExpiresAt
	refreshed.GoogleProviderIdentity = session.GoogleProviderIdentity
	if err := h.hydrateConnectedPlatforms(refreshed); err != nil {
		return nil, err
	}
	return refreshed, nil
}

func (h *AuthHandler) ensureGoogleProviderFresh(session *authSession) (*authSession, error) {
	if session == nil || !session.GoogleConnected {
		return session, nil
	}

	if session.GoogleRefreshToken == "" {
		return session, nil
	}

	if session.GoogleExpiresAt.IsZero() || time.Until(session.GoogleExpiresAt) > time.Minute {
		return session, nil
	}

	token, err := h.refreshGoogleProviderToken(session.GoogleRefreshToken)
	if err != nil {
		return nil, err
	}

	updated := *session
	updated.GoogleProviderToken = token.AccessToken
	updated.GoogleRefreshToken = firstNonEmpty(token.RefreshToken, session.GoogleRefreshToken)
	updated.GoogleTokenType = firstNonEmpty(token.TokenType, session.GoogleTokenType)
	updated.GoogleExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	if err := h.upsertProviderToken(
		updated.UserID,
		"google",
		updated.GoogleProviderToken,
		updated.GoogleRefreshToken,
		updated.GoogleTokenType,
		updated.GoogleScopes,
		updated.GoogleExpiresAt,
	); err != nil {
		return nil, err
	}

	h.mu.Lock()
	h.session = &updated
	h.mu.Unlock()

	return &updated, nil
}

func (h *AuthHandler) hydrateConnectedPlatforms(session *authSession) error {
	if session == nil || session.UserID == "" {
		return nil
	}

	records, err := h.fetchConnectedPlatforms(session.UserID)
	if err != nil {
		return err
	}

	for _, record := range records {
		if record.Provider != "google" || record.Status != "connected" {
			continue
		}

		session.GoogleConnected = true
		session.GoogleEmail = firstNonEmpty(record.ExternalEmail, session.GoogleEmail)
		session.GoogleName = firstNonEmpty(record.ExternalName, session.GoogleName)
	}

	tokenRecord, err := h.fetchProviderToken(session.UserID, "google")
	if err != nil {
		return err
	}

	if tokenRecord != nil {
		session.GoogleProviderToken = tokenRecord.AccessToken
		session.GoogleRefreshToken = tokenRecord.RefreshToken
		session.GoogleTokenType = tokenRecord.TokenType
		session.GoogleScopes = tokenRecord.Scopes
		if tokenRecord.ExpiresAt != nil {
			session.GoogleExpiresAt = *tokenRecord.ExpiresAt
		}
	}

	return nil
}

func (h *AuthHandler) fetchConnectedPlatforms(userID string) ([]connectedPlatformRecord, error) {
	var records []connectedPlatformRecord
	query := url.Values{}
	query.Set("user_id", "eq."+userID)
	query.Set("select", "user_id,provider,external_email,external_name,status")

	if err := h.supabaseAdminJSONRequest(http.MethodGet, "/connected_platforms", query, nil, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func (h *AuthHandler) upsertConnectedPlatform(userID string, provider string, externalEmail string, externalName string) error {
	if userID == "" || provider == "" {
		return nil
	}

	body := []connectedPlatformRecord{
		{
			UserID:        userID,
			Provider:      provider,
			ExternalEmail: externalEmail,
			ExternalName:  externalName,
			Status:        "connected",
		},
	}

	query := url.Values{}
	query.Set("on_conflict", "user_id,provider")

	return h.supabaseAdminJSONRequest(http.MethodPost, "/connected_platforms", query, body, nil)
}

func (h *AuthHandler) deleteConnectedPlatform(userID string, provider string) error {
	if userID == "" || provider == "" {
		return nil
	}

	query := url.Values{}
	query.Set("user_id", "eq."+userID)
	query.Set("provider", "eq."+provider)

	return h.supabaseAdminJSONRequest(http.MethodDelete, "/connected_platforms", query, nil, nil)
}

func (h *AuthHandler) fetchProviderToken(userID string, provider string) (*providerTokenRecord, error) {
	if userID == "" || provider == "" {
		return nil, nil
	}

	var records []providerTokenRecord
	query := url.Values{}
	query.Set("user_id", "eq."+userID)
	query.Set("provider", "eq."+provider)
	query.Set("select", "user_id,provider,access_token,refresh_token,token_type,scopes,expires_at")

	if err := h.supabaseAdminJSONRequest(http.MethodGet, "/provider_tokens", query, nil, &records); err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	return &records[0], nil
}

func (h *AuthHandler) upsertProviderToken(userID string, provider string, accessToken string, refreshToken string, tokenType string, scopes string, expiresAt time.Time) error {
	if userID == "" || provider == "" {
		return nil
	}

	record := providerTokenRecord{
		UserID:       userID,
		Provider:     provider,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		Scopes:       scopes,
	}
	if !expiresAt.IsZero() {
		record.ExpiresAt = &expiresAt
	}

	query := url.Values{}
	query.Set("on_conflict", "user_id,provider")

	return h.supabaseAdminJSONRequest(http.MethodPost, "/provider_tokens", query, []providerTokenRecord{record}, nil)
}

func (h *AuthHandler) deleteProviderToken(userID string, provider string) error {
	if userID == "" || provider == "" {
		return nil
	}

	query := url.Values{}
	query.Set("user_id", "eq."+userID)
	query.Set("provider", "eq."+provider)

	return h.supabaseAdminJSONRequest(http.MethodDelete, "/provider_tokens", query, nil, nil)
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

func (h *AuthHandler) exchangeGoogleCode(code string) (*googleTokenResponse, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("google did not return an authorization code")
	}

	payload := url.Values{}
	payload.Set("code", code)
	payload.Set("client_id", h.cfg.Google.ClientID)
	payload.Set("client_secret", h.cfg.Google.ClientSecret)
	payload.Set("redirect_uri", h.cfg.Google.ConnectRedirectURL)
	payload.Set("grant_type", "authorization_code")

	request, err := http.NewRequest(http.MethodPost, googleTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google token request: %w", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := h.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Google token endpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Google token response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("google token exchange failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var token googleTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to decode Google token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("google token exchange returned no access token")
	}

	return &token, nil
}

func (h *AuthHandler) refreshGoogleProviderToken(refreshToken string) (*googleTokenResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("google refresh token is missing")
	}

	payload := url.Values{}
	payload.Set("client_id", h.cfg.Google.ClientID)
	payload.Set("client_secret", h.cfg.Google.ClientSecret)
	payload.Set("refresh_token", refreshToken)
	payload.Set("grant_type", "refresh_token")

	request, err := http.NewRequest(http.MethodPost, googleTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google refresh request: %w", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := h.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Google token endpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Google refresh response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("google token refresh failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var token googleTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to decode Google refresh response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("google token refresh returned no access token")
	}

	return &token, nil
}

func (h *AuthHandler) fetchGoogleUserInfo(accessToken string) (*googleUserInfo, error) {
	request, err := http.NewRequest(http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google userinfo request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := h.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Google userinfo endpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Google userinfo response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("google userinfo request failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var userInfo googleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Google userinfo response: %w", err)
	}

	return &userInfo, nil
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

func (h *AuthHandler) supabaseAdminJSONRequest(method string, path string, query url.Values, body any, target any) error {
	if h.cfg.Supabase.URL == "" || h.cfg.Supabase.ServiceRoleKey == "" {
		return fmt.Errorf("supabase service role is not configured")
	}

	endpoint := h.cfg.Supabase.URL + supabaseRESTPath + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var payload io.Reader
	if body != nil {
		buffer, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode Supabase admin request body: %w", err)
		}
		payload = strings.NewReader(string(buffer))
	}

	request, err := http.NewRequest(method, endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to create Supabase admin request: %w", err)
	}

	request.Header.Set("apikey", h.cfg.Supabase.ServiceRoleKey)
	request.Header.Set("Authorization", "Bearer "+h.cfg.Supabase.ServiceRoleKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Prefer", "return=minimal,resolution=merge-duplicates")

	response, err := h.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to reach Supabase admin api: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read Supabase admin response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = "unknown Supabase admin error"
		}
		return fmt.Errorf("supabase admin request failed with HTTP %d: %s", response.StatusCode, message)
	}

	if target == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("failed to decode Supabase admin response: %w", err)
	}

	return nil
}

func newSessionFromSupabase(authProvider string, accessToken string, refreshToken string, tokenType string, expiresIn int, user *supabaseUser) *authSession {
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	if expiresIn <= 0 {
		expiresAt = time.Time{}
	}

	return &authSession{
		UserID:       user.ID,
		Email:        user.Email,
		Name:         pickDisplayName(user),
		AuthProvider: authProvider,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		ExpiresAt:    expiresAt,
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

func normalizeGoogleIntent(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "connect":
		return "connect"
	default:
		return "login"
	}
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
        fetch("/auth/session", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ ...payload, provider: "google" })
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

func oauthRedirectBodyHTML(provider string, intent string, status string, message string) string {
	callbackURL := fmt.Sprintf(
		"kai://auth/callback?provider=%s&intent=%s&status=%s&message=%s",
		url.QueryEscape(provider),
		url.QueryEscape(intent),
		url.QueryEscape(status),
		url.QueryEscape(message),
	)

	return fmt.Sprintf(`<p id="status">%s</p>
    <p class="muted" id="detail">%s</p>
    <script>
      const callbackUrl = %q;
      window.location.replace(callbackUrl);
      setTimeout(() => {
        const statusNode = document.getElementById("status");
        const detailNode = document.getElementById("detail");
        statusNode.textContent = %q;
        detailNode.textContent = "If Kai did not reopen automatically, return to the app manually.";
      }, 1200);
    </script>`,
		map[bool]string{true: "Google connected", false: "Google connection failed"}[status == "success"],
		message,
		callbackURL,
		map[bool]string{true: "Still waiting for Kai to reopen...", false: "Kai did not reopen automatically."}[status == "success"],
	)
}
