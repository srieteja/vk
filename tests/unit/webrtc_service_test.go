package unit

import (
	"fmt"
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
	if parsedToken.UserID != 1 {
		t.Errorf("expected user ID 1 embedded in token, got %d", parsedToken.UserID)
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

func TestVerifyMembership(t *testing.T) {
	db := setupTestDB()
	service := services.NewWebRTCService(db, "secret")

	call := models.Call{CallerID: 1, ReceiverID: 2, Status: "initiated"}
	db.Create(&call)
	callIDStr := fmt.Sprintf("%d", call.ID)

	if err := service.VerifyMembership(callIDStr, 1); err != nil {
		t.Errorf("expected caller to pass membership check, got %v", err)
	}
	if err := service.VerifyMembership(callIDStr, 2); err != nil {
		t.Errorf("expected receiver to pass membership check, got %v", err)
	}
	if err := service.VerifyMembership(callIDStr, 99); err == nil {
		t.Error("expected non-member to fail membership check")
	}
	if err := service.VerifyMembership("999999", 1); err == nil {
		t.Error("expected nonexistent call to fail membership check")
	}
}

func TestVerifyMembership_InactiveCallRejected(t *testing.T) {
	db := setupTestDB()
	service := services.NewWebRTCService(db, "secret")

	call := models.Call{CallerID: 1, ReceiverID: 2, Status: "completed"}
	db.Create(&call)

	if err := service.VerifyMembership(fmt.Sprintf("%d", call.ID), 1); err == nil {
		t.Error("expected membership check to fail for a completed call")
	}
}
