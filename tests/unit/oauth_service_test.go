package unit

import (
	"strings"
	"testing"

	"vk_backend/internal/config"
	"vk_backend/internal/services"
)

func TestGenerateState(t *testing.T) {
	db := setupTestDB()
	cfg := &config.Config{
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-secret",
		GoogleRedirectURL:  "http://localhost:8080/callback",
	}
	authService := services.NewAuthService(db)
	service := services.NewOAuthService(db, cfg, authService)

	state := service.GenerateState("advocate")
	if !strings.HasPrefix(state, "advocate_") {
		t.Errorf("expected state to start with 'advocate_', got %s", state)
	}

	state2 := service.GenerateState("client")
	if !strings.HasPrefix(state2, "client_") {
		t.Errorf("expected state to start with 'client_', got %s", state2)
	}

	if state == state2 {
		t.Error("expected unique states")
	}
}

func TestGetAuthURL(t *testing.T) {
	db := setupTestDB()
	cfg := &config.Config{
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-secret",
		GoogleRedirectURL:  "http://localhost:8080/callback",
	}
	authService := services.NewAuthService(db)
	service := services.NewOAuthService(db, cfg, authService)

	state := "test_state"
	url := service.GetAuthURL(state)

	if !strings.Contains(url, "https://accounts.google.com/o/oauth2/auth") {
		t.Errorf("expected google auth url, got %s", url)
	}
	if !strings.Contains(url, "client_id=test-client-id") {
		t.Errorf("expected client_id in url, got %s", url)
	}
	if !strings.Contains(url, "state=test_state") {
		t.Errorf("expected state in url, got %s", url)
	}
}
