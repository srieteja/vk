package services

import (
	"encoding/json"
)

type WebRTCService struct{}

func NewWebRTCService() *WebRTCService {
	return &WebRTCService{}
}

func (s *WebRTCService) GenerateToken(callID string) (string, error) {
	token := map[string]string{
		"call_id": callID,
		"type":    "webrtc",
	}

	data, _ := json.Marshal(token)
	return string(data), nil
}