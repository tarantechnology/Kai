package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port     string
	Supabase SupabaseConfig
	Google   GoogleConfig
}

type SupabaseConfig struct {
	URL              string
	AnonKey          string
	ServiceRoleKey   string
	RedirectURL      string
	EmailRedirectURL string
}

type GoogleConfig struct {
	ClientID           string
	ClientSecret       string
	Scopes             []string
	ConnectRedirectURL string
}

func Load() Config {
	loadDotEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	supabase := loadSupabaseConfig()
	google := loadGoogleConfig()

	return Config{
		Port:     port,
		Supabase: supabase,
		Google:   google,
	}
}

func loadSupabaseConfig() SupabaseConfig {
	redirectURL := strings.TrimSpace(os.Getenv("SUPABASE_REDIRECT_URL"))

	emailRedirectURL := strings.TrimSpace(os.Getenv("SUPABASE_EMAIL_REDIRECT_URL"))

	return SupabaseConfig{
		URL:              strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_URL")), "/"),
		AnonKey:          strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY")),
		ServiceRoleKey:   strings.TrimSpace(os.Getenv("SUPABASE_SERVICE_ROLE_KEY")),
		RedirectURL:      defaultIfEmpty(redirectURL, "kai://auth/callback"),
		EmailRedirectURL: defaultIfEmpty(emailRedirectURL, "kai://auth/callback"),
	}
}

func loadGoogleConfig() GoogleConfig {
	scopes := strings.Split(strings.TrimSpace(os.Getenv("GOOGLE_SCOPES")), ",")
	filteredScopes := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			filteredScopes = append(filteredScopes, scope)
		}
	}

	return GoogleConfig{
		ClientID:           strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		ClientSecret:       strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		Scopes:             filteredScopes,
		ConnectRedirectURL: defaultIfEmpty(strings.TrimSpace(os.Getenv("GOOGLE_CONNECT_REDIRECT_URL")), "http://127.0.0.1:8080/auth/google/connect/callback"),
	}
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

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
