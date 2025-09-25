package config

import (
	"errors"
	"os"
)

type ThemeConfig struct {
	LogoURL      string
	PrimaryColor string
	FontSans     string
}

type Config struct {
	DBPath              string
	ContentPath         string
	GitRepoURL          string
	GitSSHKeyPath       string
	GithubWebhookSecret string
	AdminSecret         string
	AdminLoginPath      string
	Theme               ThemeConfig
}

func Load() (*Config, error) {
	repoURL := os.Getenv("GIT_REPO_URL")
	if repoURL == "" {
		return nil, errors.New("GIT_REPO_URL environment variable is required")
	}

	keyPath := os.Getenv("GIT_SSH_KEY_PATH")
	if keyPath == "" {
		return nil, errors.New("GIT_SSH_KEY_PATH environment variable is required")
	}

	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if webhookSecret == "" {
		return nil, errors.New("GITHUB_WEBHOOK_SECRET environment variable is required")
	}

	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret == "" {
		return nil, errors.New("ADMIN_SECRET environment variable is required")
	}

	adminLoginPath := os.Getenv("ADMIN_LOGIN_PATH")
	if adminLoginPath == "" {
		adminLoginPath = "/admin-login" // Default value
	}

	return &Config{
		DBPath:              "./gmd-data.db",
		ContentPath:         "./content",
		GitRepoURL:          repoURL,
		GitSSHKeyPath:       keyPath,
		GithubWebhookSecret: webhookSecret,
		AdminSecret:         adminSecret,
		AdminLoginPath:      adminLoginPath,
		Theme: ThemeConfig{
			LogoURL:      getEnv("THEME_LOGO_URL", "/static/img/logo.svg"),
			PrimaryColor: getEnv("THEME_COLOR_PRIMARY", "#3498db"),
			FontSans:     getEnv("THEME_FONT_SANS", "Inter"),
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
