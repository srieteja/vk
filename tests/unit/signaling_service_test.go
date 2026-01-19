package unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vk_backend/internal/models"
	"vk_backend/internal/services"

	"github.com/gorilla/websocket"
)

func TestSignalingFlow(t *testing.T) {
	db := setupTestDB()
	secret := "test-secret"
	webrtcService := services.NewWebRTCService(db, secret)
	signalingService := services.NewSignalingService(webrtcService)

	// Setup Test Server
	s := httptest.NewServer(http.HandlerFunc(signalingService.HandleWebSocket))
	defer s.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	// Create valid Token
	call := models.Call{
		CallerID:     1,
		ReceiverID:   2,
		Status:       "initiated",
		CallerType:   "client",
		ReceiverType: "advocate",
	}
	db.Create(&call)
	token, _ := webrtcService.GenerateToken(call.ID, 1)

	// Connect Peer A
	peerA, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Peer A failed to connect: %v", err)
	}
	defer peerA.Close()

	// Authenticate Peer A
	if err := peerA.WriteJSON(map[string]string{"token": token}); err != nil {
		t.Fatalf("Peer A failed to auth: %v", err)
	}

	// Connect Peer B
	peerB, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Peer B failed to connect: %v", err)
	}
	defer peerB.Close()

	// Authenticate Peer B (Using same token for simplicity, usually would be distinct)
	// In real world, Peer B would have their own token for the same Call ID
	if err := peerB.WriteJSON(map[string]string{"token": token}); err != nil {
		t.Fatalf("Peer B failed to auth: %v", err)
	}

	// Test Message Exchange (A -> B)
	msg := services.SignalingMessage{
		Type: "offer",
		// Payload would be SDP
	}

	if err := peerA.WriteJSON(msg); err != nil {
		t.Fatalf("Peer A failed to send message: %v", err)
	}

	var receivedMsg services.SignalingMessage
	if err := peerB.ReadJSON(&receivedMsg); err != nil {
		t.Fatalf("Peer B failed to read message: %v", err)
	}

	if receivedMsg.Type != "offer" {
		t.Errorf("expected type 'offer', got %s", receivedMsg.Type)
	}
}
