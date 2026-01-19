package unit

import (
	"testing"
	"vk_backend/internal/services"
)

func TestInitiateCall(t *testing.T) {
	db := setupTestDB()
	service := services.NewCallService(db)

	call, err := service.InitiateCall(1, "client", 2, "advocate")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if call.CallerID != 1 {
		t.Errorf("expected caller ID 1, got %d", call.CallerID)
	}

	if call.Status != "initiated" {
		t.Errorf("expected status initiated, got %s", call.Status)
	}
}

func TestAcceptCall(t *testing.T) {
	db := setupTestDB()
	service := services.NewCallService(db)

	// First initiate a call
	createdCall, err := service.InitiateCall(1, "client", 2, "advocate")
	if err != nil {
		t.Fatalf("failed to initiate call: %v", err)
	}

	// Then accept it as the receiver
	call, err := service.AcceptCall(createdCall.ID, 2)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if call.Status != "accepted" {
		t.Errorf("expected status accepted, got %s", call.Status)
	}
}

func TestEndCall(t *testing.T) {
	db := setupTestDB()
	service := services.NewCallService(db)

	// First initiate
	createdCall, err := service.InitiateCall(1, "client", 2, "advocate")
	if err != nil {
		t.Fatalf("failed to initiate call: %v", err)
	}

	// Then accept
	_, err = service.AcceptCall(createdCall.ID, 2)
	if err != nil {
		t.Fatalf("failed to accept call: %v", err)
	}

	// Then end it (as either party, let's say caller)
	call, err := service.EndCall(createdCall.ID, 1)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if call.Status != "completed" {
		t.Errorf("expected status completed, got %s", call.Status)
	}
}

func TestGetCall(t *testing.T) {
	db := setupTestDB()
	service := services.NewCallService(db)

	createdCall, _ := service.InitiateCall(1, "client", 2, "advocate")

	// Caller retrieves call
	call, err := service.GetCall(createdCall.ID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if call.ID != createdCall.ID {
		t.Error("retrieved wrong call")
	}

	// Unauthorized user retrieves call
	_, err = service.GetCall(createdCall.ID, 99)
	if err == nil {
		t.Error("expected error for unauthorized user")
	}
}

func TestAcceptCall_Unauthorized(t *testing.T) {
	db := setupTestDB()
	service := services.NewCallService(db)

	createdCall, _ := service.InitiateCall(1, "client", 2, "advocate")

	// User 3 tries to accept call intended for User 2
	_, err := service.AcceptCall(createdCall.ID, 3)
	if err == nil {
		t.Error("expected error for unauthorized acceptance")
	}
}

func TestEndCall_AlreadyCompleted(t *testing.T) {
	db := setupTestDB()
	service := services.NewCallService(db)

	createdCall, _ := service.InitiateCall(1, "client", 2, "advocate")
	service.AcceptCall(createdCall.ID, 2)
	service.EndCall(createdCall.ID, 1)

	// Try ending again
	_, err := service.EndCall(createdCall.ID, 1)
	if err == nil {
		t.Error("expected error for ending completed call")
	}
}
