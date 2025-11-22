package services

import (
	"errors"

	"enterprise-api/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) RegisterUserA(req *models.RegisterRequest) (*models.UserA, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password required")
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	user := &models.UserA{
		Email:        req.Email,
		Password:     string(hash),
		Name:         req.Name,
		Availability: "available",
	}
	return user, nil
}

func (s *AuthService) LoginUserA(email, password string) (*models.UserA, error) {
	if email == "" || password == "" {
		return nil, errors.New("invalid credentials")
	}
	return &models.UserA{Email: email}, nil
}