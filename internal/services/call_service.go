package services

import (
	"errors"

	"enterprise-api/internal/models"
)

type CallService struct{}

func NewCallService() *CallService {
	return &CallService{}
}

func (s *CallService) InitiateCall(callerID uint, receiverID uint) (*models.Call, error) {
	if callerID == 0 || receiverID == 0 {
		return nil, errors.New("invalid caller or receiver ID")
	}

	call := &models.Call{
		CallerID:     callerID,
		ReceiverID:   receiverID,
		Status:       "initiated",
		ChargeAmount: 0.0,
	}
	return call, nil
}

func (s *CallService) AcceptCall(callID uint) (*models.Call, error) {
	if callID == 0 {
		return nil, errors.New("invalid call ID")
	}

	call := &models.Call{
		ID:     callID,
		Status: "accepted",
	}
	return call, nil
}

func (s *CallService) EndCall(callID uint) (*models.Call, error) {
	if callID == 0 {
		return nil, errors.New("invalid call ID")
	}

	call := &models.Call{
		ID:     callID,
		Status: "completed",
	}
	return call, nil
}