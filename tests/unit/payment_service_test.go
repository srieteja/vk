package unit

import (
	"testing"

	"vk_backend/internal/config"
	"vk_backend/internal/models"
	"vk_backend/internal/services"
)

func setupTestConfig() *config.Config {
	return &config.Config{
		PlatformCommission: 20.0, // 20% commission
	}
}

func TestInitiatePayment(t *testing.T) {
	db := setupTestDB()
	cfg := setupTestConfig()
	service := services.NewPaymentService(db, cfg, nil)

	// Create test users
	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid", Balance: 100}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid"}
	db.Create(&client)
	db.Create(&advocate)

	amount := 50.00
	payment, err := service.InitiatePayment(client.ID, advocate.ID, amount)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payment.Amount != amount {
		t.Errorf("expected amount %f, got %f", amount, payment.Amount)
	}

	expectedFee := amount * 0.20
	if payment.PlatformFee != expectedFee {
		t.Errorf("expected platform fee %f, got %f", expectedFee, payment.PlatformFee)
	}

	expectedCommission := amount - expectedFee
	if payment.AdvocateCommission != expectedCommission {
		t.Errorf("expected advocate commission %f, got %f", expectedCommission, payment.AdvocateCommission)
	}

	if payment.Status != "pending" {
		t.Errorf("expected status pending, got %s", payment.Status)
	}
}

func TestVerifyPayment(t *testing.T) {
	db := setupTestDB()
	cfg := setupTestConfig()
	service := services.NewPaymentService(db, cfg, nil)

	// Create test users
	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid", Balance: 100}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", Earnings: 0}
	db.Create(&client)
	db.Create(&advocate)

	// Setup: Create a payment first
	amount := 100.00
	p, err := service.InitiatePayment(client.ID, advocate.ID, amount)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	payment, err := service.VerifyPayment(p.TransactionID, client.ID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payment.Status != "completed" {
		t.Errorf("expected status completed, got %s", payment.Status)
	}

	// Verify advocate earnings updated
	var updatedAdvocate models.Advocate
	db.First(&updatedAdvocate, advocate.ID)
	expectedEarnings := amount * 0.80 // 80% of 100
	if updatedAdvocate.Earnings != expectedEarnings {
		t.Errorf("expected advocate earnings %f, got %f", expectedEarnings, updatedAdvocate.Earnings)
	}

	// Verify duplicate verification doesn't double-credit earnings
	_, err = service.VerifyPayment(p.TransactionID, client.ID)
	if err != nil {
		t.Fatalf("expected no error on duplicate verification, got %v", err)
	}

	db.First(&updatedAdvocate, advocate.ID)
	if updatedAdvocate.Earnings != expectedEarnings {
		t.Errorf("expected advocate earnings to remain %f, got %f", expectedEarnings, updatedAdvocate.Earnings)
	}
}

func TestVerifyPaymentWrongOwnerReturnsSameErrorAsNotFound(t *testing.T) {
	db := setupTestDB()
	cfg := setupTestConfig()
	service := services.NewPaymentService(db, cfg, nil)

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	otherClient := models.Client{Name: "Other", Email: "other@test.com", UUID: "other-uuid"}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid"}
	db.Create(&client)
	db.Create(&otherClient)
	db.Create(&advocate)

	p, err := service.InitiatePayment(client.ID, advocate.ID, 50.00)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, wrongOwnerErr := service.VerifyPayment(p.TransactionID, otherClient.ID)
	_, notFoundErr := service.VerifyPayment("txn_does_not_exist", otherClient.ID)

	if wrongOwnerErr == nil {
		t.Fatal("expected an error when verifying someone else's transaction")
	}
	if wrongOwnerErr.Error() != notFoundErr.Error() {
		t.Errorf("wrong-owner and not-found errors must be identical to avoid enumeration, got %q vs %q",
			wrongOwnerErr.Error(), notFoundErr.Error())
	}
}

func TestInitiatePayment_InvalidAmount(t *testing.T) {
	db := setupTestDB()
	cfg := setupTestConfig()
	service := services.NewPaymentService(db, cfg, nil)

	_, err := service.InitiatePayment(1, 2, 0)

	if err == nil {
		t.Fatalf("expected error for zero amount")
	}
}
