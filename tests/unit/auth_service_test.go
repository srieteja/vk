package unit

import (
	"testing"

	"enterprise-api/internal/models"
	"enterprise-api/internal/services"
)

func TestRegisterUserA(t *testing.T) {
	service := services.NewAuthService()

	req := &models.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	user, err := service.RegisterUserA(req)

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
	service := services.NewAuthService()

	user, err := service.LoginUserA("test@example.com", "password123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email, got %s", user.Email)
	}
}

func TestRegisterUserA_InvalidEmail(t *testing.T) {
	service := services.NewAuthService()

	req := &models.RegisterRequest{
		Email:    "",
		Password: "password123",
		Name:     "Test",
	}

	_, err := service.RegisterUserA(req)

	if err == nil {
		t.Fatalf("expected error for empty email")
	}
}