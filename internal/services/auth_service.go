package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"enterprise-api/internal/logger"
	"enterprise-api/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db:     db,
		logger: logger.NewLogger("AuthService", logger.INFO),
	}
}

func (s *AuthService) RegisterAdvocate(req *models.RegisterRequest) (*models.Advocate, error) {
	s.logger.Finer("RegisterAdvocate called for email: %s", req.Email)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		s.logger.Info("RegisterAdvocate failed: missing required fields")
		return nil, errors.New("email, password, and name are required")
	}

	// Check if email already exists
	var existingAdvocate models.Advocate
	if err := s.db.Where("email = ?", req.Email).First(&existingAdvocate).Error; err == nil {
		s.logger.Info("RegisterAdvocate failed: email already exists: %s", req.Email)
		return nil, errors.New("email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Severe("RegisterAdvocate failed: password hashing error: %v", err)
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
		s.logger.Severe("RegisterAdvocate failed: database error: %v", err)
		return nil, errors.New("failed to create advocate")
	}

	s.logger.Info("Advocate registered successfully: ID=%d, Email=%s", advocate.ID, advocate.Email)
	return advocate, nil
}

func (s *AuthService) LoginAdvocate(email, password string) (*models.Advocate, error) {
	s.logger.Finer("LoginAdvocate called for email: %s", email)

	if email == "" || password == "" {
		s.logger.Info("LoginAdvocate failed: missing credentials")
		return nil, errors.New("email and password are required")
	}

	var advocate models.Advocate
	if err := s.db.Where("email = ?", email).First(&advocate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("LoginAdvocate failed: advocate not found for email: %s", email)
			return nil, errors.New("invalid credentials")
		}
		s.logger.Severe("LoginAdvocate failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(advocate.Password), []byte(password)); err != nil {
		s.logger.Info("LoginAdvocate failed: invalid password for email: %s", email)
		return nil, errors.New("invalid credentials")
	}

	s.logger.Info("Advocate logged in successfully: ID=%d, Email=%s", advocate.ID, email)
	return &advocate, nil
}

func (s *AuthService) RegisterClient(req *models.RegisterRequest) (*models.Client, error) {
	s.logger.Finer("RegisterClient called for email: %s", req.Email)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		s.logger.Info("RegisterClient failed: missing required fields")
		return nil, errors.New("email, password, and name are required")
	}

	// Check if email already exists
	var existingClient models.Client
	if err := s.db.Where("email = ?", req.Email).First(&existingClient).Error; err == nil {
		s.logger.Info("RegisterClient failed: email already exists: %s", req.Email)
		return nil, errors.New("email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Severe("RegisterClient failed: password hashing error: %v", err)
		return nil, errors.New("failed to hash password")
	}

	// Create client
	client := &models.Client{
		Email:    req.Email,
		Password: string(hash),
		Name:     req.Name,
		Balance:  0.0,
	}

	if err := s.db.Create(client).Error; err != nil {
		s.logger.Severe("RegisterClient failed: database error: %v", err)
		return nil, errors.New("failed to create client")
	}

	s.logger.Info("Client registered successfully: ID=%d, Email=%s", client.ID, client.Email)
	return client, nil
}

func (s *AuthService) LoginClient(email, password string) (*models.Client, error) {
	s.logger.Finer("LoginClient called for email: %s", email)

	if email == "" || password == "" {
		s.logger.Info("LoginClient failed: missing credentials")
		return nil, errors.New("email and password are required")
	}

	var client models.Client
	if err := s.db.Where("email = ?", email).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("LoginClient failed: client not found for email: %s", email)
			return nil, errors.New("invalid credentials")
		}
		s.logger.Severe("LoginClient failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(client.Password), []byte(password)); err != nil {
		s.logger.Info("LoginClient failed: invalid password for email: %s", email)
		return nil, errors.New("invalid credentials")
	}

	s.logger.Info("Client logged in successfully: ID=%d, Email=%s", client.ID, email)
	return &client, nil
}

func (s *AuthService) CreateSession(userID uint, userType string) (*models.Session, error) {
	s.logger.Finer("CreateSession called for userID=%d, userType=%s", userID, userType)

	// Generate a secure token (in production, use JWT or similar)
	token, err := generateToken()
	if err != nil {
		s.logger.Severe("CreateSession failed: token generation error: %v", err)
		return nil, errors.New("failed to generate session token")
	}

	session := &models.Session{
		UserID:    userID,
		UserType:  userType,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour expiry
	}

	if err := s.db.Create(session).Error; err != nil {
		s.logger.Severe("CreateSession failed: database error: %v", err)
		return nil, errors.New("failed to create session")
	}

	s.logger.Finer("Session created successfully: UserID=%d, UserType=%s", userID, userType)
	return session, nil
}

func generateToken() (string, error) {
	// Generate a secure random token
	// If random number generation fails, we must fail the operation
	// rather than using a predictable fallback that compromises security
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
