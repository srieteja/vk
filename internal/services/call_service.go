package services

import (
	"errors"
	"time"

	"enterprise-api/internal/logger"
	"enterprise-api/internal/models"

	"gorm.io/gorm"
)

type CallService struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewCallService(db *gorm.DB) *CallService {
	return &CallService{
		db:     db,
		logger: logger.NewLogger("CallService", logger.INFO),
	}
}

func (s *CallService) InitiateCall(callerID uint, callerType string, receiverID uint, receiverType string) (*models.Call, error) {
	s.logger.Finer("InitiateCall called: callerID=%d, callerType=%s, receiverID=%d, receiverType=%s", callerID, callerType, receiverID, receiverType)

	if callerID == 0 || receiverID == 0 {
		s.logger.Info("InitiateCall failed: invalid caller or receiver ID")
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
		s.logger.Severe("InitiateCall failed: database error: %v", err)
		return nil, errors.New("failed to create call")
	}

	s.logger.Info("Call initiated successfully: ID=%d, CallerID=%d, ReceiverID=%d", call.ID, call.CallerID, call.ReceiverID)
	return call, nil
}

func (s *CallService) AcceptCall(callID uint, userID uint) (*models.Call, error) {
	s.logger.Finer("AcceptCall called: callID=%d, userID=%d", callID, userID)

	if callID == 0 {
		s.logger.Info("AcceptCall failed: invalid call ID")
		return nil, errors.New("invalid call ID")
	}

	var call models.Call
	if err := s.db.First(&call, callID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("AcceptCall failed: call not found: callID=%d", callID)
			return nil, errors.New("call not found")
		}
		s.logger.Severe("AcceptCall failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	// Verify user is the receiver of this call
	if call.ReceiverID != userID {
		s.logger.Info("AcceptCall failed: user not authorized: callID=%d, userID=%d, receiverID=%d", callID, userID, call.ReceiverID)
		return nil, errors.New("unauthorized: you are not the receiver of this call")
	}

	// Check if call is in valid state to accept
	if call.Status != "initiated" {
		s.logger.Info("AcceptCall failed: invalid call status: callID=%d, status=%s", callID, call.Status)
		return nil, errors.New("call cannot be accepted in current status")
	}

	now := time.Now()
	call.Status = "accepted"
	call.StartedAt = &now

	if err := s.db.Save(&call).Error; err != nil {
		s.logger.Severe("AcceptCall failed: failed to update call: %v", err)
		return nil, errors.New("failed to update call")
	}

	s.logger.Info("Call accepted successfully: ID=%d, ReceiverID=%d", call.ID, call.ReceiverID)
	return &call, nil
}

func (s *CallService) EndCall(callID uint, userID uint) (*models.Call, error) {
	s.logger.Finer("EndCall called: callID=%d, userID=%d", callID, userID)

	if callID == 0 {
		s.logger.Info("EndCall failed: invalid call ID")
		return nil, errors.New("invalid call ID")
	}

	var call models.Call
	if err := s.db.First(&call, callID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("EndCall failed: call not found: callID=%d", callID)
			return nil, errors.New("call not found")
		}
		s.logger.Severe("EndCall failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	// Verify user is part of this call (either caller or receiver)
	if call.CallerID != userID && call.ReceiverID != userID {
		s.logger.Info("EndCall failed: user not authorized: callID=%d, userID=%d", callID, userID)
		return nil, errors.New("unauthorized: you are not part of this call")
	}

	// Check if call is in valid state to end
	if call.Status == "completed" {
		s.logger.Info("EndCall failed: call already completed: callID=%d", callID)
		return nil, errors.New("call is already completed")
	}

	now := time.Now()
	call.Status = "completed"
	call.EndedAt = &now

	if call.StartedAt != nil {
		call.Duration = int64(now.Sub(*call.StartedAt).Seconds())
	}

	if err := s.db.Save(&call).Error; err != nil {
		s.logger.Severe("EndCall failed: failed to update call: %v", err)
		return nil, errors.New("failed to update call")
	}

	s.logger.Info("Call ended successfully: ID=%d, Duration=%d seconds", call.ID, call.Duration)
	return &call, nil
}

// GetCall retrieves a call by ID and verifies user access
func (s *CallService) GetCall(callID uint, userID uint) (*models.Call, error) {
	s.logger.Finer("GetCall called: callID=%d, userID=%d", callID, userID)

	if callID == 0 {
		return nil, errors.New("invalid call ID")
	}

	var call models.Call
	if err := s.db.First(&call, callID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("GetCall failed: call not found: callID=%d", callID)
			return nil, errors.New("call not found")
		}
		s.logger.Severe("GetCall failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	// Verify user is part of this call
	if call.CallerID != userID && call.ReceiverID != userID {
		s.logger.Info("GetCall failed: user not authorized: callID=%d, userID=%d", callID, userID)
		return nil, errors.New("unauthorized: you are not part of this call")
	}

	return &call, nil
}
