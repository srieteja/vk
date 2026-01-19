package unit

import (
	"testing"
	"time"

	"vk_backend/internal/models"
	"vk_backend/internal/services"
)

func TestGetAdvocateSchedule(t *testing.T) {
	db := setupTestDB()
	service := services.NewAdvocateService(db)

	// Create test advocate
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid"}
	db.Create(&advocate)

	// Create some calls
	now := time.Now()
	later := now.Add(2 * time.Hour)

	calls := []models.Call{
		{
			CallerID:     100, // Random client ID
			ReceiverID:   advocate.ID,
			ReceiverType: "advocate",
			Status:       "scheduled",
			ScheduledAt:  &later,
		},
		{
			CallerID:     101,
			ReceiverID:   advocate.ID,
			ReceiverType: "advocate",
			Status:       "completed", // Should be ignored
			StartedAt:    &now,
		},
	}
	db.Create(&calls)

	// Test getting schedule
	schedule, err := service.GetAdvocateSchedule(advocate.ID, "", "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(schedule) != 1 {
		t.Errorf("expected 1 scheduled call, got %d", len(schedule))
	}

	if schedule[0].Status != "scheduled" {
		t.Errorf("expected status scheduled, got %s", schedule[0].Status)
	}
}

func TestUpdateAvailability(t *testing.T) {
	db := setupTestDB()
	service := services.NewAdvocateService(db)

	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", Availability: "offline"}
	db.Create(&advocate)

	updated, err := service.UpdateAvailability(advocate.ID, "available")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Availability != "available" {
		t.Errorf("expected availability available, got %s", updated.Availability)
	}

	// Verify DB update
	var dbAdvocate models.Advocate
	db.First(&dbAdvocate, advocate.ID)
	if dbAdvocate.Availability != "available" {
		t.Error("database not updated")
	}
}

func TestGetProfile(t *testing.T) {
	db := setupTestDB()
	service := services.NewAdvocateService(db)

	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com"}
	db.Create(&advocate)

	profile, err := service.GetProfile(advocate.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.Email != "a@test.com" {
		t.Errorf("expected email a@test.com, got %s", profile.Email)
	}
}

func TestGetEarnings(t *testing.T) {
	db := setupTestDB()
	service := services.NewAdvocateService(db)

	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", Earnings: 100.0}
	db.Create(&advocate)

	// Create a completed call for this advocate
	db.Create(&models.Call{ReceiverID: advocate.ID, ReceiverType: "advocate", Status: "completed"})

	earnings, err := service.GetEarnings(advocate.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if earnings["earnings"] != 100.0 {
		t.Errorf("expected earnings 100.0, got %v", earnings["earnings"])
	}

	if earnings["total_calls"] != int64(1) {
		t.Errorf("expected 1 total call, got %v", earnings["total_calls"])
	}
}

func TestUpdateProfileImage(t *testing.T) {
	db := setupTestDB()
	service := services.NewAdvocateService(db)

	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com"}
	db.Create(&advocate)

	updated, err := service.UpdateProfileImage(advocate.ID, "http://image.com/pic.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.ProfileImage != "http://image.com/pic.jpg" {
		t.Errorf("expected updated image, got %s", updated.ProfileImage)
	}
}

func TestGetAvailableAdvocates(t *testing.T) {
	db := setupTestDB()
	service := services.NewAdvocateService(db)

	db.Create(&models.Advocate{Name: "A1", Email: "a1@test.com", UUID: "uuid-1", Availability: "available", Location: "NY", HourlyRate: 50})
	db.Create(&models.Advocate{Name: "A2", Email: "a2@test.com", UUID: "uuid-2", Availability: "offline", Location: "CA", HourlyRate: 60})
	db.Create(&models.Advocate{Name: "A3", Email: "a3@test.com", UUID: "uuid-3", Availability: "available", Location: "NY", HourlyRate: 100})

	// Test filter by location and availability
	filter := &services.AdvocateFilter{
		Availability: "available",
		Location:     "ny",
	}

	advocates, err := service.GetAvailableAdvocates(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(advocates) != 2 {
		t.Errorf("expected 2 advocates, got %d", len(advocates))
	}
}
