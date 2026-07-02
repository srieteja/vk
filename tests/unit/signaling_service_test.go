package unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vk_backend/internal/models"
	"vk_backend/internal/services"

	"github.com/gorilla/websocket"
)

func TestSignalingFlow(t *testing.T) {
	db := setupTestDB()
	secret := "test-secret"
	webrtcService := services.NewWebRTCService(db, secret)
	signalingService := services.NewSignalingService(webrtcService, nil, nil)

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
	if err := db.Create(&call).Error; err != nil {
		t.Fatalf("failed to create call: %v", err)
	}
	token, err := webrtcService.GenerateToken(call.ID, 1)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Connect Peer A
	peerA, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Peer A failed to connect: %v", err)
	}
	defer func() {
		_ = peerA.Close()
	}()

	// Authenticate Peer A
	if err := peerA.WriteJSON(map[string]string{"token": token}); err != nil {
		t.Fatalf("Peer A failed to auth: %v", err)
	}

	// Connect Peer B
	peerB, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Peer B failed to connect: %v", err)
	}
	defer func() {
		_ = peerB.Close()
	}()

	// Authenticate Peer B (Using same token for simplicity, usually would be distinct)
	// In real world, Peer B would have their own token for the same Call ID
	if err := peerB.WriteJSON(map[string]string{"token": token}); err != nil {
		t.Fatalf("Peer B failed to auth: %v", err)
	}

	// Peer B's auth (including the server-side membership DB check) happens
	// asynchronously on its own connection goroutine; give it time to finish
	// registering before Peer A broadcasts, or the message would be dropped
	// with no recipient in the room yet.
	time.Sleep(200 * time.Millisecond)

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

func TestSignalingRejectsThirdPeer(t *testing.T) {
	db := setupTestDB()
	secret := "test-secret"
	webrtcService := services.NewWebRTCService(db, secret)
	signalingService := services.NewSignalingService(webrtcService, nil, nil)

	s := httptest.NewServer(http.HandlerFunc(signalingService.HandleWebSocket))
	defer s.Close()
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	call := models.Call{
		CallerID:     1,
		ReceiverID:   2,
		Status:       "initiated",
		CallerType:   "client",
		ReceiverType: "advocate",
	}
	if err := db.Create(&call).Error; err != nil {
		t.Fatalf("failed to create call: %v", err)
	}

	callerToken, err := webrtcService.GenerateToken(call.ID, 1)
	if err != nil {
		t.Fatalf("failed to generate caller token: %v", err)
	}
	receiverToken, err := webrtcService.GenerateToken(call.ID, 2)
	if err != nil {
		t.Fatalf("failed to generate receiver token: %v", err)
	}

	connectAndAuth := func(token string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		if err := conn.WriteJSON(map[string]string{"token": token}); err != nil {
			t.Fatalf("failed to auth: %v", err)
		}
		return conn
	}

	peerA := connectAndAuth(callerToken)
	defer func() { _ = peerA.Close() }()
	peerB := connectAndAuth(receiverToken)
	defer func() { _ = peerB.Close() }()

	// Give the server a moment to register both peers before the third join
	// attempt, since addClient runs in each connection's own goroutine.
	time.Sleep(200 * time.Millisecond)

	// A third connection (even with a validly-signed token for a real
	// member) must be rejected: the room is already full.
	thirdToken, err := webrtcService.GenerateToken(call.ID, 1)
	if err != nil {
		t.Fatalf("failed to generate third token: %v", err)
	}
	peerC, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("peer C failed to connect at transport level: %v", err)
	}
	defer func() { _ = peerC.Close() }()
	if err := peerC.WriteJSON(map[string]string{"token": thirdToken}); err != nil {
		t.Fatalf("peer C failed to send auth message: %v", err)
	}

	var resp map[string]string
	if err := peerC.ReadJSON(&resp); err != nil {
		t.Fatalf("expected a room-full error response, got read error: %v", err)
	}
	if resp["error"] == "" {
		t.Errorf("expected an error response for the third peer, got %v", resp)
	}
}

func TestSignalingRejectsDisallowedOrigin(t *testing.T) {
	db := setupTestDB()
	secret := "test-secret"
	webrtcService := services.NewWebRTCService(db, secret)
	signalingService := services.NewSignalingService(webrtcService, []string{"http://allowed.example"}, nil)

	server := httptest.NewServer(http.HandlerFunc(signalingService.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	headers := http.Header{}
	headers.Set("Origin", "http://evil.example")

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, headers)
	if err == nil {
		if conn != nil {
			_ = conn.Close()
		}
		t.Fatal("expected connection to be rejected for disallowed origin")
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
}
