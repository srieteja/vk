package services

import (
	"errors"
	"strings"

	"vk_backend/internal/logger"
	"vk_backend/internal/models"
	"vk_backend/internal/validators"

	"gorm.io/gorm"
)

type AdvocateService struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewAdvocateService(db *gorm.DB) *AdvocateService {
	return &AdvocateService{
		db:     db,
		logger: logger.NewLogger("AdvocateService", logger.INFO),
	}
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

func (s *AdvocateService) UpdateProfileImage(advocateID uint, imageURL string) (*models.Advocate, error) {
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

	advocate.ProfileImage = imageURL
	if err := s.db.Save(&advocate).Error; err != nil {
		return nil, errors.New("failed to update profile image")
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
	if err := s.db.Model(&models.Call{}).Where("receiver_id = ? AND receiver_type = ? AND status = ?", advocateID, "advocate", "completed").Count(&totalCalls).Error; err != nil {
		s.logger.Severe("GetEarnings failed: error counting calls for advocate ID=%d: %v", advocateID, err)
		return nil, errors.New("failed to count calls")
	}

	s.logger.Finer("GetEarnings: advocate ID=%d, earnings=%.2f, total_calls=%d", advocateID, advocate.Earnings, totalCalls)
	return map[string]interface{}{
		"earnings":    advocate.Earnings,
		"total_calls": totalCalls,
		"hourly_rate": advocate.HourlyRate,
	}, nil
}

type AdvocateFilter struct {
	Availability string  // "all" or "available" (online)
	Location     string  // City/location filter
	MinRate      float64 // Minimum hourly rate
	MaxRate      float64 // Maximum hourly rate
}

func (s *AdvocateService) GetAvailableAdvocates(filter *AdvocateFilter) ([]models.Advocate, error) {
	query := s.db.Model(&models.Advocate{})

	// Availability filter
	if filter != nil && filter.Availability != "" {
		if filter.Availability == "available" || filter.Availability == "online" {
			query = query.Where("availability = ?", "available")
		}
		// If "all", don't filter by availability
	} else {
		// Default: only show available advocates
		query = query.Where("availability = ?", "available")
	}

	// Location filter
	if filter != nil && filter.Location != "" {
		escaped := validators.EscapeLikePattern(strings.ToLower(filter.Location))
		query = query.Where("LOWER(location) LIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}

	// Rate range filter
	if filter != nil && filter.MinRate > 0 {
		query = query.Where("hourly_rate >= ?", filter.MinRate)
	}
	if filter != nil && filter.MaxRate > 0 {
		query = query.Where("hourly_rate <= ?", filter.MaxRate)
	}

	var advocates []models.Advocate
	if err := query.Find(&advocates).Error; err != nil {
		s.logger.Severe("GetAvailableAdvocates failed: %v", err)
		return nil, errors.New("failed to fetch advocates")
	}

	return advocates, nil
}

func (s *AdvocateService) GetAdvocateSchedule(advocateID uint, from, to string) ([]models.Call, error) {
	if advocateID == 0 {
		return nil, errors.New("invalid advocate ID")
	}

	query := s.db.Where("receiver_id = ? AND receiver_type = ?", advocateID, "advocate").
		Where("status IN ?", []string{"scheduled", "accepted", "initiated"})

	// Filter by time range if provided
	if from != "" {
		query = query.Where("scheduled_at >= ? OR started_at >= ?", from, from)
	}
	if to != "" {
		query = query.Where("scheduled_at <= ? OR started_at <= ?", to, to)
	}

	var calls []models.Call
	if err := query.Order("scheduled_at asc, started_at asc").Find(&calls).Error; err != nil {
		s.logger.Severe("GetAdvocateSchedule failed: %v", err)
		return nil, errors.New("failed to fetch schedule")
	}

	return calls, nil
}
