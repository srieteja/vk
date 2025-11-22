package unit

import (
	"testing"

	"enterprise-api/internal/services"
)

func TestInitiatePayment(t *testing.T) {
	service := services.NewPaymentService()

	payment, err := service.InitiatePayment(1, 2, 50.00)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payment.Amount != 50.00 {
		t.Errorf("expected amount 50.00, got %f", payment.Amount)
	}

	if payment.Status != "pending" {
		t.Errorf("expected status pending, got %s", payment.Status)
	}
}

func TestVerifyPayment(t *testing.T) {
	service := services.NewPaymentService()

	payment, err := service.VerifyPayment("txn_123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payment.Status != "completed" {
		t.Errorf("expected status completed, got %s", payment.Status)
	}
}

func TestInitiatePayment_InvalidAmount(t *testing.T) {
	service := services.NewPaymentService()

	_, err := service.InitiatePayment(1, 2, 0)

	if err == nil {
		t.Fatalf("expected error for zero amount")
	}
}