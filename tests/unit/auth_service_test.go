package unit

import (
	"testing"

	"vk_backend/internal/models"
	"vk_backend/internal/services"
)

func TestRegisterUserA(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db)

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
	service := services.NewAuthService(db)

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
	service := services.NewAuthService(db)

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
	service := services.NewAuthService(db)

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
	service := services.NewAuthService(db)

	req := &models.RegisterRequest{
		Email:    "client@test.com",
		Password: "password123",
		Name:     "Test Client",
	}
	service.RegisterClient(req)

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
	service := services.NewAuthService(db)

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
