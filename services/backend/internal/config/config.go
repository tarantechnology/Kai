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
}

type SupabaseConfig struct {
	URL              string
	AnonKey          string
	ServiceRoleKey   string
	RedirectURL      string
	EmailRedirectURL string
}

func Load() Config {
	loadDotEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	supabase := loadSupabaseConfig()

	return Config{
		Port:     port,
		Supabase: supabase,
	}
}

func loadSupabaseConfig() SupabaseConfig {
	redirectURL := strings.TrimSpace(os.Getenv("SUPABASE_REDIRECT_URL"))

	emailRedirectURL := strings.TrimSpace(os.Getenv("SUPABASE_EMAIL_REDIRECT_URL"))

	return SupabaseConfig{
		URL:              strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_URL")), "/"),
		AnonKey:          strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY")),
		ServiceRoleKey:   strings.TrimSpace(os.Getenv("SUPABASE_SERVICE_ROLE_KEY")),
		RedirectURL:      redirectURL,
		EmailRedirectURL: emailRedirectURL,
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
