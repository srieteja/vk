package unit

import (
	"context"
	"testing"

	"vk_backend/internal/models"
	"vk_backend/internal/services"
	"vk_backend/internal/sessions"
)

func TestRegisterUserA(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	req := &models.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	user, err := service.RegisterAdvocate(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}

	if user.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", user.Name)
	}
}

func TestLoginUserA(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	// Need to register first
	req := &models.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, err := service.RegisterAdvocate(req)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	user, err := service.LoginAdvocate("test@example.com", "password123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email, got %s", user.Email)
	}
}

func TestRegisterUserA_InvalidEmail(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	req := &models.RegisterRequest{
		Email:    "",
		Password: "password123",
		Name:     "Test",
	}

	_, err := service.RegisterAdvocate(req)

	if err == nil {
		t.Fatalf("expected error for empty email")
	}
}

func TestRegisterClient(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	req := &models.RegisterRequest{
		Email:    "client@test.com",
		Password: "password123",
		Name:     "Test Client",
	}

	client, err := service.RegisterClient(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.Email != "client@test.com" {
		t.Errorf("expected email client@test.com, got %s", client.Email)
	}
}

func TestLoginClient(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	req := &models.RegisterRequest{
		Email:    "client@test.com",
		Password: "password123",
		Name:     "Test Client",
	}
	if _, err := service.RegisterClient(req); err != nil {
		t.Fatalf("expected no error registering client, got %v", err)
	}

	client, err := service.LoginClient("client@test.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.Email != "client@test.com" {
		t.Errorf("expected email client@test.com, got %s", client.Email)
	}
}

func TestCreateSession(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	session, err := service.CreateSession(1, "advocate")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if session.Token == "" {
		t.Error("expected token to be generated")
	}

	if session.UserID != 1 {
		t.Errorf("expected user ID 1, got %d", session.UserID)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	db := setupTestDB()
	store := setupSessionStore(db)
	service := services.NewAuthService(db, store)

	session, err := service.CreateSession(1, "advocate")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if _, err := store.GetByToken(context.Background(), session.Token); err != nil {
		t.Fatalf("session should exist before logout: %v", err)
	}

	if err := service.Logout(context.Background(), session.Token); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := store.GetByToken(context.Background(), session.Token); err != sessions.ErrSessionNotFound {
		t.Errorf("expected session to be revoked, got err=%v", err)
	}
}

func TestLogoutIsIdempotent(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	if err := service.Logout(context.Background(), "never-existed"); err != nil {
		t.Errorf("expected logout of unknown token to be a no-op, got %v", err)
	}
}

func TestRevokeUserSessionsRemovesAllTokens(t *testing.T) {
	db := setupTestDB()
	store := setupSessionStore(db)
	service := services.NewAuthService(db, store)

	s1, err := service.CreateSession(42, "advocate")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	s2, err := service.CreateSession(42, "advocate")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	other, err := service.CreateSession(99, "client")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := service.RevokeUserSessions(context.Background(), 42); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, tok := range []string{s1.Token, s2.Token} {
		if _, err := store.GetByToken(context.Background(), tok); err != sessions.ErrSessionNotFound {
			t.Errorf("expected session %q to be revoked, got err=%v", tok, err)
		}
	}

	if _, err := store.GetByToken(context.Background(), other.Token); err != nil {
		t.Errorf("expected unrelated user's session to survive, got err=%v", err)
	}
}
