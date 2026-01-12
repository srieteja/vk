package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"enterprise-api/internal/config"
	"enterprise-api/internal/logger"
	"enterprise-api/internal/models"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type OAuthService struct {
	db          *gorm.DB
	logger      *logger.Logger
	oauthConfig *oauth2.Config
	authService *AuthService
}

func NewOAuthService(db *gorm.DB, cfg *config.Config, authService *AuthService) *OAuthService {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &OAuthService{
		db:          db,
		logger:      logger.NewLogger("OAuthService", logger.INFO),
		oauthConfig: oauthConfig,
		authService: authService,
	}
}

func (s *OAuthService) GetAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (s *OAuthService) HandleCallback(code string, userType string) (*models.Session, error) {
	s.logger.Finer("Handling OAuth callback for userType: %s", userType)

	// Exchange code for token
	token, err := s.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		s.logger.Severe("OAuth token exchange failed: %v", err)
		return nil, errors.New("failed to exchange token")
	}

	// Get user info from Google
	client := s.oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		s.logger.Severe("Failed to get user info: %v", err)
		return nil, errors.New("failed to get user info")
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		s.logger.Severe("Failed to decode user info: %v", err)
		return nil, errors.New("failed to decode user info")
	}

	// Find or create user
	var userID uint
	if userType == "advocate" {
		advocate, err := s.findOrCreateAdvocate(userInfo)
		if err != nil {
			return nil, err
		}
		userID = advocate.ID
	} else if userType == "client" {
		client, err := s.findOrCreateClient(userInfo)
		if err != nil {
			return nil, err
		}
		userID = client.ID
	} else {
		return nil, errors.New("invalid user type")
	}

	// Create session
	session, err := s.authService.CreateSession(userID, userType)
	if err != nil {
		s.logger.Severe("Failed to create session: %v", err)
		return nil, errors.New("failed to create session")
	}

	return session, nil
}

func (s *OAuthService) findOrCreateAdvocate(userInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}) (*models.Advocate, error) {
	var advocate models.Advocate

	// Try to find by email
	err := s.db.Where("email = ?", userInfo.Email).First(&advocate).Error
	if err == nil {
		// User exists, update profile image if provided
		if userInfo.Picture != "" && advocate.ProfileImage == "" {
			advocate.ProfileImage = userInfo.Picture
			s.db.Save(&advocate)
		}
		// Update GoogleID if not set
		if advocate.GoogleID == nil {
			advocate.GoogleID = &userInfo.ID
			s.db.Save(&advocate)
		}
		s.logger.Info("Advocate found via OAuth: ID=%d, Email=%s", advocate.ID, advocate.Email)
		return &advocate, nil
	}

	if err != gorm.ErrRecordNotFound {
		s.logger.Severe("Database error finding advocate: %v", err)
		return nil, errors.New("database error")
	}

	// Create new advocate
	advocate = models.Advocate{
		Email:        userInfo.Email,
		GoogleID:     &userInfo.ID,
		Password:     "", // OAuth users don't have passwords
		Name:         userInfo.Name,
		Availability: "available",
		UUID:         uuid.New().String(),
		ProfileImage: userInfo.Picture,
		Earnings:     0.0,
		HourlyRate:   0.0,
	}

	if err := s.db.Create(&advocate).Error; err != nil {
		s.logger.Severe("Failed to create advocate: %v", err)
		return nil, errors.New("failed to create advocate")
	}

	s.logger.Info("Advocate created via OAuth: ID=%d, Email=%s", advocate.ID, advocate.Email)
	return &advocate, nil
}

func (s *OAuthService) findOrCreateClient(userInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}) (*models.Client, error) {
	var client models.Client

	// Try to find by email
	err := s.db.Where("email = ?", userInfo.Email).First(&client).Error
	if err == nil {
		// User exists, update profile image if provided
		if userInfo.Picture != "" && client.ProfileImage == "" {
			client.ProfileImage = userInfo.Picture
			s.db.Save(&client)
		}
		// Update GoogleID if not set
		if client.GoogleID == nil {
			client.GoogleID = &userInfo.ID
			s.db.Save(&client)
		}
		s.logger.Info("Client found via OAuth: ID=%d, Email=%s", client.ID, client.Email)
		return &client, nil
	}

	if err != gorm.ErrRecordNotFound {
		s.logger.Severe("Database error finding client: %v", err)
		return nil, errors.New("database error")
	}

	// Create new client
	client = models.Client{
		Email:        userInfo.Email,
		GoogleID:     &userInfo.ID,
		Password:     "", // OAuth users don't have passwords
		Name:         userInfo.Name,
		UUID:         uuid.New().String(),
		ProfileImage: userInfo.Picture,
		Balance:      0.0,
	}

	if err := s.db.Create(&client).Error; err != nil {
		s.logger.Severe("Failed to create client: %v", err)
		return nil, errors.New("failed to create client")
	}

	s.logger.Info("Client created via OAuth: ID=%d, Email=%s", client.ID, client.Email)
	return &client, nil
}

func (s *OAuthService) GenerateState(userType string) string {
	// Generate a simple state token (in production, use a more secure method)
	return fmt.Sprintf("%s_%s", userType, uuid.New().String())
}

