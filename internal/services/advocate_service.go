package services

import (
	"errors"

	"enterprise-api/internal/models"

	"gorm.io/gorm"
)

type AdvocateService struct {
	db *gorm.DB
}

func NewAdvocateService(db *gorm.DB) *AdvocateService {
	return &AdvocateService{db: db}
}

func (s *AdvocateService) UpdateAvailability(advocateID uint, availability string) (*models.Advocate, error) {
	if advocateID == 0 {
		return nil, errors.New("invalid advocate ID")
	}

	validStatuses := map[string]bool{
		"available": true,
		"busy":      true,
		"offline":   true,
	}
	if !validStatuses[availability] {
		return nil, errors.New("invalid availability status")
	}

	var advocate models.Advocate
	if err := s.db.First(&advocate, advocateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("advocate not found")
		}
		return nil, errors.New("database error")
	}

	advocate.Availability = availability
	if err := s.db.Save(&advocate).Error; err != nil {
		return nil, errors.New("failed to update availability")
	}

	return &advocate, nil
}

func (s *AdvocateService) GetProfile(advocateID uint) (*models.Advocate, error) {
	if advocateID == 0 {
		return nil, errors.New("invalid advocate ID")
	}

	var advocate models.Advocate
	if err := s.db.First(&advocate, advocateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("advocate not found")
		}
		return nil, errors.New("database error")
	}

	return &advocate, nil
}

func (s *AdvocateService) GetEarnings(advocateID uint) (map[string]interface{}, error) {
	if advocateID == 0 {
		return nil, errors.New("invalid advocate ID")
	}

	var advocate models.Advocate
	if err := s.db.First(&advocate, advocateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("advocate not found")
		}
		return nil, errors.New("database error")
	}

	// Count total calls
	var totalCalls int64
	s.db.Model(&models.Call{}).Where("receiver_id = ? AND receiver_type = ? AND status = ?", advocateID, "advocate", "completed").Count(&totalCalls)

	return map[string]interface{}{
		"earnings":    advocate.Earnings,
		"total_calls": totalCalls,
		"hourly_rate": advocate.HourlyRate,
	}, nil
}

func (s *AdvocateService) GetAvailableAdvocates() ([]models.Advocate, error) {
	var advocates []models.Advocate
	if err := s.db.Where("availability = ?", "available").Find(&advocates).Error; err != nil {
		return nil, errors.New("failed to fetch advocates")
	}

	return advocates, nil
}

