package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vk_backend/internal/logger"
	"vk_backend/internal/models"

	"gorm.io/gorm"
)

type WebRTCService struct {
	db     *gorm.DB
	logger *logger.Logger
	secret string
}

type WebRTCToken struct {
	CallID    string `json:"call_id"`
	Type      string `json:"type"`
	ExpiresAt int64  `json:"expires_at"`
	Signature string `json:"signature"`
}

func NewWebRTCService(db *gorm.DB, secret string) *WebRTCService {
	return &WebRTCService{
		db:     db,
		logger: logger.NewLogger("WebRTCService", logger.INFO),
		secret: secret,
	}
}

// GenerateToken generates a secure WebRTC token for a call
func (s *WebRTCService) GenerateToken(callID uint, userID uint) (string, error) {
	s.logger.Finer("GenerateToken called for callID=%d, userID=%d", callID, userID)

	// Validate call exists and user is authorized
	var call models.Call
	if err := s.db.First(&call, callID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("GenerateToken failed: call not found: callID=%d", callID)
			return "", errors.New("call not found")
		}
		s.logger.Severe("GenerateToken failed: database error: %v", err)
		return "", errors.New("database error")
	}

	// Verify user is part of this call
	if call.CallerID != userID && call.ReceiverID != userID {
		s.logger.Info("GenerateToken failed: user not authorized for call: callID=%d, userID=%d", callID, userID)
		return "", errors.New("unauthorized access to call")
	}

	// Check call status
	if call.Status != "initiated" && call.Status != "accepted" {
		s.logger.Info("GenerateToken failed: invalid call status: callID=%d, status=%s", callID, call.Status)
		return "", errors.New("call is not active")
	}

	// Create token with expiration (1 hour)
	// Convert callID to string for token storage
	expiresAt := time.Now().Add(1 * time.Hour).Unix()
	token := WebRTCToken{
		CallID:    fmt.Sprintf("%d", callID),
		Type:      "webrtc",
		ExpiresAt: expiresAt,
	}

	// Create signature
	signature, err := s.createSignature(token)
	if err != nil {
		s.logger.Severe("GenerateToken failed: signature creation error: %v", err)
		return "", errors.New("failed to create token signature")
	}
	token.Signature = signature

	// Marshal to JSON
	data, err := json.Marshal(token)
	if err != nil {
		s.logger.Severe("GenerateToken failed: JSON marshal error: %v", err)
		return "", errors.New("failed to marshal token")
	}

	// Encode to base64 for transport
	encodedToken := base64.URLEncoding.EncodeToString(data)

	s.logger.Info("WebRTC token generated successfully: callID=%d, userID=%d", callID, userID)
	return encodedToken, nil
}

// ValidateToken validates a WebRTC token
func (s *WebRTCService) ValidateToken(tokenString string) (*WebRTCToken, error) {
	s.logger.Finer("ValidateToken called")

	// Decode from base64
	data, err := base64.URLEncoding.DecodeString(tokenString)
	if err != nil {
		s.logger.Info("ValidateToken failed: invalid base64 encoding")
		return nil, errors.New("invalid token format")
	}

	// Unmarshal JSON
	var token WebRTCToken
	if err := json.Unmarshal(data, &token); err != nil {
		s.logger.Info("ValidateToken failed: invalid JSON: %v", err)
		return nil, errors.New("invalid token format")
	}

	// Check expiration
	if time.Now().Unix() > token.ExpiresAt {
		s.logger.Info("ValidateToken failed: token expired: callID=%s", token.CallID)
		return nil, errors.New("token expired")
	}

	// Verify signature
	expectedSig, err := s.createSignature(token)
	if err != nil {
		s.logger.Severe("ValidateToken failed: signature creation error: %v", err)
		return nil, errors.New("failed to validate token")
	}

	if !hmac.Equal([]byte(token.Signature), []byte(expectedSig)) {
		s.logger.Info("ValidateToken failed: invalid signature: callID=%s", token.CallID)
		return nil, errors.New("invalid token signature")
	}

	s.logger.Finer("Token validated successfully: callID=%s", token.CallID)
	return &token, nil
}

// createSignature creates HMAC signature for the token
func (s *WebRTCService) createSignature(token WebRTCToken) (string, error) {
	// Create a copy without signature for signing
	signToken := WebRTCToken{
		CallID:    token.CallID,
		Type:      token.Type,
		ExpiresAt: token.ExpiresAt,
	}

	data, err := json.Marshal(signToken)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write(data)
	signature := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	return signature, nil
}

// GetICEServers returns STUN/TURN server configuration for WebRTC
func (s *WebRTCService) GetICEServers() []map[string]interface{} {
	// Default STUN servers (can be configured via environment)
	// In production, you should use your own TURN servers
	return []map[string]interface{}{
		{
			"urls": []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
			},
		},
	}
}
