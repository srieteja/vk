package unit

import (
	"testing"

	"vk_backend/internal/models"
	"vk_backend/internal/services"
)

func TestGenerateAndValidateToken(t *testing.T) {
	db := setupTestDB()
	secret := "test-secret"
	service := services.NewWebRTCService(db, secret)

	// Setup call and user
	call := models.Call{
		CallerID:     1,
		ReceiverID:   2,
		Status:       "initiated",
		CallerType:   "client",
		ReceiverType: "advocate",
	}
	db.Create(&call)

	// Test GenerateToken
	token, err := service.GenerateToken(call.ID, 1)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken returned empty string")
	}

	// Test ValidateToken
	parsedToken, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if parsedToken.CallID != "1" {
		t.Errorf("expected call ID 1, got %s", parsedToken.CallID)
	}
}

func TestGenerateToken_InvalidCall(t *testing.T) {
	db := setupTestDB()
	service := services.NewWebRTCService(db, "secret")

	_, err := service.GenerateToken(999, 1)
	if err == nil {
		t.Fatal("expected error for invalid call ID")
	}
}

func TestGenerateToken_Unauthorized(t *testing.T) {
	db := setupTestDB()
	service := services.NewWebRTCService(db, "secret")

	call := models.Call{
		CallerID:     1,
		ReceiverID:   2,
		Status:       "initiated",
		CallerType:   "client",
		ReceiverType: "advocate",
	}
	db.Create(&call)

	_, err := service.GenerateToken(call.ID, 3) // User 3 is not part of call
	if err == nil {
		t.Fatal("expected error for unauthorized user")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	db := setupTestDB()
	service := services.NewWebRTCService(db, "secret")
	otherService := services.NewWebRTCService(db, "wrong-secret")

	call := models.Call{CallerID: 1, ReceiverID: 2, Status: "initiated"}
	db.Create(&call)

	token, _ := service.GenerateToken(call.ID, 1)

	_, err := otherService.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}
