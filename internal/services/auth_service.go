package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"enterprise-api/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) RegisterAdvocate(req *models.RegisterRequest) (*models.Advocate, error) {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, errors.New("email, password, and name are required")
	}

	// Check if email already exists
	var existingAdvocate models.Advocate
	if err := s.db.Where("email = ?", req.Email).First(&existingAdvocate).Error; err == nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create advocate
	advocate := &models.Advocate{
		Email:        req.Email,
		Password:     string(hash),
		Name:         req.Name,
		Availability: "available",
		Earnings:     0.0,
		HourlyRate:   0.0,
	}

	if err := s.db.Create(advocate).Error; err != nil {
		return nil, errors.New("failed to create advocate")
	}

	return advocate, nil
}

func (s *AuthService) LoginAdvocate(email, password string) (*models.Advocate, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	var advocate models.Advocate
	if err := s.db.Where("email = ?", email).First(&advocate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("invalid credentials")
		}
		return nil, errors.New("database error")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(advocate.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &advocate, nil
}

func (s *AuthService) RegisterClient(req *models.RegisterRequest) (*models.Client, error) {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, errors.New("email, password, and name are required")
	}

	// Check if email already exists
	var existingClient models.Client
	if err := s.db.Where("email = ?", req.Email).First(&existingClient).Error; err == nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create client
	client := &models.Client{
		Email:   req.Email,
		Password: string(hash),
		Name:    req.Name,
		Balance: 0.0,
	}

	if err := s.db.Create(client).Error; err != nil {
		return nil, errors.New("failed to create client")
	}

	return client, nil
}

func (s *AuthService) LoginClient(email, password string) (*models.Client, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	var client models.Client
	if err := s.db.Where("email = ?", email).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("invalid credentials")
		}
		return nil, errors.New("database error")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(client.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &client, nil
}

func (s *AuthService) CreateSession(userID uint, userType string) (*models.Session, error) {
	// Generate a simple token (in production, use JWT or similar)
	token := generateToken()
	
	session := &models.Session{
		UserID:    userID,
		UserType:  userType,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour expiry
	}

	if err := s.db.Create(session).Error; err != nil {
		return nil, errors.New("failed to create session")
	}

	return session, nil
}

func generateToken() string {
	// Generate a secure random token
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
