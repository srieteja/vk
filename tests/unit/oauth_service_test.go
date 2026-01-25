package unit

import (
	"strings"
	"testing"
	"time"

	"vk_backend/internal/config"
	"vk_backend/internal/services"
)

func TestGenerateState(t *testing.T) {
	db := setupTestDB()
	cfg := &config.Config{
		GoogleClientID:       "test-client-id",
		GoogleClientSecret:   "test-secret",
		GoogleRedirectURL:    "http://localhost:8080/callback",
		OAuthStateSecret:     "state-secret",
		OAuthStateTTLSeconds: 60,
	}
	authService := services.NewAuthService(db, setupSessionStore(db))
	service := services.NewOAuthService(db, cfg, authService)

	state := service.GenerateState("advocate")
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	userType, err := service.ValidateState(state)
	if err != nil {
		t.Fatalf("expected valid state, got %v", err)
	}
	if userType != "advocate" {
		t.Errorf("expected userType advocate, got %s", userType)
	}

	state2 := service.GenerateState("client")
	if state2 == "" {
		t.Fatal("expected non-empty state")
	}

	if state == state2 {
		t.Error("expected unique states")
	}
}

func TestGetAuthURL(t *testing.T) {
	db := setupTestDB()
	cfg := &config.Config{
		GoogleClientID:       "test-client-id",
		GoogleClientSecret:   "test-secret",
		GoogleRedirectURL:    "http://localhost:8080/callback",
		OAuthStateSecret:     "state-secret",
		OAuthStateTTLSeconds: 60,
	}
	authService := services.NewAuthService(db, setupSessionStore(db))
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

func TestValidateState_Tampered(t *testing.T) {
	db := setupTestDB()
	cfg := &config.Config{
		GoogleClientID:       "test-client-id",
		GoogleClientSecret:   "test-secret",
		GoogleRedirectURL:    "http://localhost:8080/callback",
		OAuthStateSecret:     "state-secret",
		OAuthStateTTLSeconds: 60,
	}
	authService := services.NewAuthService(db, setupSessionStore(db))
	service := services.NewOAuthService(db, cfg, authService)

	state := service.GenerateState("advocate")
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	tampered := state + "x"
	if _, err := service.ValidateState(tampered); err == nil {
		t.Fatal("expected error for tampered state")
	}
}

func TestValidateState_Expired(t *testing.T) {
	db := setupTestDB()
	cfg := &config.Config{
		GoogleClientID:       "test-client-id",
		GoogleClientSecret:   "test-secret",
		GoogleRedirectURL:    "http://localhost:8080/callback",
		OAuthStateSecret:     "state-secret",
		OAuthStateTTLSeconds: 1,
	}
	authService := services.NewAuthService(db, setupSessionStore(db))
	service := services.NewOAuthService(db, cfg, authService)

	state := service.GenerateState("client")
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	time.Sleep(2 * time.Second)
	if _, err := service.ValidateState(state); err == nil {
		t.Fatal("expected expired state error")
	}
}
