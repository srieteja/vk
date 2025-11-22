package services

import (
	"errors"
	"fmt"

	"enterprise-api/internal/models"
)

type PaymentService struct{}

func NewPaymentService() *PaymentService {
	return &PaymentService{}
}

func (s *PaymentService) InitiatePayment(userBID, userAID uint, amount float64) (*models.Payment, error) {
	if amount <= 0 {
		return nil, errors.New("invalid amount")
	}

	payment := &models.Payment{
		UserBID:        userBID,
		UserAID:        userAID,
		Amount:         amount,
		Status:         "pending",
		TransactionID:  fmt.Sprintf("txn_%d", userBID),
		PaymentGateway: "stripe",
	}
	return payment, nil
}

func (s *PaymentService) VerifyPayment(txnID string) (*models.Payment, error) {
	if txnID == "" {
		return nil, errors.New("invalid transaction ID")
	}

	payment := &models.Payment{
		TransactionID: txnID,
		Status:        "completed",
	}
	return payment, nil
}