package config

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port   string
	Google GoogleOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type googleSecretFile struct {
	Installed *googleClientCredentials `json:"installed"`
	Web       *googleClientCredentials `json:"web"`
}

type googleClientCredentials struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURIs []string `json:"redirect_uris"`
}

func Load() Config {
	loadDotEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	google := loadGoogleOAuthConfig()

	return Config{
		Port:   port,
		Google: google,
	}
}

func loadGoogleOAuthConfig() GoogleOAuthConfig {
	cfg := GoogleOAuthConfig{
		ClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		RedirectURL:  strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")),
		Scopes:       loadGoogleScopes(),
	}

	secretFilePath := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET_FILE"))
	if secretFilePath == "" {
		secretFilePath = "kaiWebCreds.json"
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
		loadGoogleConfigFromFile(secretFilePath, &cfg)
	}

	if cfg.RedirectURL == "" {
		cfg.RedirectURL = "http://127.0.0.1:8080/auth/google/callback"
	}

	return cfg
}

func loadDotEnv() {
	candidates := []string{
		".env",
		filepath.Join("services", "backend", ".env"),
	}

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}

			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)

			if key == "" {
				continue
			}

			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}

		return
	}
}

func loadGoogleScopes() []string {
	raw := strings.TrimSpace(os.Getenv("GOOGLE_SCOPES"))
	if raw == "" {
		return []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/calendar",
		}
	}

	parts := strings.Split(raw, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope != "" {
			scopes = append(scopes, scope)
		}
	}

	return scopes
}

func loadGoogleConfigFromFile(path string, cfg *GoogleOAuthConfig) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var parsed googleSecretFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return
	}

	credentials := parsed.Installed
	if credentials == nil {
		credentials = parsed.Web
	}
	if credentials == nil {
		return
	}

	if cfg.ClientID == "" {
		cfg.ClientID = credentials.ClientID
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = credentials.ClientSecret
	}
	if cfg.RedirectURL == "" && len(credentials.RedirectURIs) > 0 {
		cfg.RedirectURL = credentials.RedirectURIs[0]
	}
}
