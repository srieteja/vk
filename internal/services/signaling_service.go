package services

import (
	"encoding/json"
	"net/http"
	"sync"

	"vk_backend/internal/logger"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now (adjust for production)
	},
}

// SignalingMessage represents the JSON payload exchanged between peers
type SignalingMessage struct {
	Type    string          `json:"type"`              // "offer", "answer", "candidate", "join"
	Payload json.RawMessage `json:"payload,omitempty"` // The SDP or ICE candidate
	Target  string          `json:"target,omitempty"`  // "peer" (default)
}

type Client struct {
	conn   *websocket.Conn
	userID uint
	callID string
}

type SignalingService struct {
	logger        *logger.Logger
	webrtcService *WebRTCService
	// Maps CallID -> List of connected Clients
	// Using a simple map for now; for horizontal scaling, use Redis Pub/Sub
	calls map[string][]*Client
	mu    sync.RWMutex
}

func NewSignalingService(webrtcService *WebRTCService) *SignalingService {
	return &SignalingService{
		logger:        logger.NewLogger("SignalingService", logger.INFO),
		webrtcService: webrtcService,
		calls:         make(map[string][]*Client),
	}
}

// HandleWebSocket upgrades the connection and starts the message loop
func (s *SignalingService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Severe("WebSocket upgrade failed: %v", err)
		return
	}

	// 2. Initial Handshake: Expect Authentication Token
	// We read the first message which MUST contain the auth token
	var initMsg struct {
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&initMsg); err != nil {
		s.logger.Info("Failed to read auth token: %v", err)
		conn.Close()
		return
	}

	// 3. Validate Token
	tokenData, err := s.webrtcService.ValidateToken(initMsg.Token)
	if err != nil {
		s.logger.Info("Invalid signaling token: %v", err)
		conn.WriteJSON(map[string]string{"error": "unauthorized"})
		conn.Close()
		return
	}

	callID := tokenData.CallID
	// We currently don't have userID in the token struct (it just has CallID, Type, ExpiresAt)
	// Ideally, the token should contain UserID to identify the sender.
	// For now, we will proceed by just grouping by CallID.
	// TODO: Update WebRTCToken to include UserID for better identification

	client := &Client{
		conn:   conn,
		callID: callID,
	}

	// 4. Register Client
	s.addClient(callID, client)
	s.logger.Info("Peer connected to call %s", callID)

	// 5. Message Loop
	defer func() {
		s.removeClient(callID, client)
		conn.Close()
	}()

	for {
		var msg SignalingMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Info("WebSocket closed unexpectedly: %v", err)
			}
			break
		}

		s.broadcastToCall(callID, client, msg)
	}
}

func (s *SignalingService) addClient(callID string, client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[callID] = append(s.calls[callID], client)
}

func (s *SignalingService) removeClient(callID string, client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	peers := s.calls[callID]
	for i, peer := range peers {
		if peer == client {
			s.calls[callID] = append(peers[:i], peers[i+1:]...)
			break
		}
	}
	if len(s.calls[callID]) == 0 {
		delete(s.calls, callID)
	}
}

// broadcastToCall sends the message to OTHER peers in the same call
func (s *SignalingService) broadcastToCall(callID string, sender *Client, msg SignalingMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers := s.calls[callID]
	for _, peer := range peers {
		if peer != sender {
			if err := peer.conn.WriteJSON(msg); err != nil {
				s.logger.Info("Failed to send message to peer: %v", err)
				// We don't remove here; let the Read loop handle disconnection errors
			}
		}
	}
}
