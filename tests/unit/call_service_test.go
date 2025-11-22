package unit

import (
	"testing"

	"enterprise-api/internal/services"
)

func TestInitiateCall(t *testing.T) {
	service := services.NewCallService()

	call, err := service.InitiateCall(1, 2)

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
	service := services.NewCallService()

	call, err := service.AcceptCall(1)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if call.Status != "accepted" {
		t.Errorf("expected status accepted, got %s", call.Status)
	}
}

func TestEndCall(t *testing.T) {
	service := services.NewCallService()

	call, err := service.EndCall(1)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if call.Status != "completed" {
		t.Errorf("expected status completed, got %s", call.Status)
	}
}