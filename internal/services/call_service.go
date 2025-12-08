package services

import (
	"errors"
	"time"

	"enterprise-api/internal/models"

	"gorm.io/gorm"
)

type CallService struct {
	db *gorm.DB
}

func NewCallService(db *gorm.DB) *CallService {
	return &CallService{db: db}
}

func (s *CallService) InitiateCall(callerID uint, callerType string, receiverID uint, receiverType string) (*models.Call, error) {
	if callerID == 0 || receiverID == 0 {
		return nil, errors.New("invalid caller or receiver ID")
	}

	call := &models.Call{
		CallerID:     callerID,
		CallerType:   callerType,
		ReceiverID:   receiverID,
		ReceiverType: receiverType,
		Status:       "initiated",
		ChargeAmount: 0.0,
	}

	if err := s.db.Create(call).Error; err != nil {
		return nil, errors.New("failed to create call")
	}

	return call, nil
}

func (s *CallService) AcceptCall(callID uint) (*models.Call, error) {
	if callID == 0 {
		return nil, errors.New("invalid call ID")
	}

	var call models.Call
	if err := s.db.First(&call, callID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("call not found")
		}
		return nil, errors.New("database error")
	}

	now := time.Now()
	call.Status = "accepted"
	call.StartedAt = &now

	if err := s.db.Save(&call).Error; err != nil {
		return nil, errors.New("failed to update call")
	}

	return &call, nil
}

func (s *CallService) EndCall(callID uint) (*models.Call, error) {
	if callID == 0 {
		return nil, errors.New("invalid call ID")
	}

	var call models.Call
	if err := s.db.First(&call, callID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("call not found")
		}
		return nil, errors.New("database error")
	}

	now := time.Now()
	call.Status = "completed"
	call.EndedAt = &now

	if call.StartedAt != nil {
		call.Duration = int64(now.Sub(*call.StartedAt).Seconds())
	}

	if err := s.db.Save(&call).Error; err != nil {
		return nil, errors.New("failed to update call")
	}

	return &call, nil
}
