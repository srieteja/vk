package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"vk_backend/internal/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// SignalingMessage represents the JSON payload exchanged between peers
type SignalingMessage struct {
	Type    string          `json:"type"`              // "offer", "answer", "candidate", "join"
	Payload json.RawMessage `json:"payload,omitempty"` // The SDP or ICE candidate
	Target  string          `json:"target,omitempty"`  // "peer" (default)
}

// maxPeersPerCall bounds a call's signaling room to the two participants
// (caller and receiver); a third join attempt (e.g. a leaked/replayed
// token) is rejected rather than silently added to the broadcast group.
const maxPeersPerCall = 2

type Client struct {
	conn   *websocket.Conn
	userID uint
	callID string
}

type SignalingService struct {
	logger          *logger.Logger
	webrtcService   *WebRTCService
	upgrader        websocket.Upgrader
	allowedOrigins  map[string]struct{}
	allowAllOrigins bool
	redisClient     *redis.Client
	subscriptions   map[string]*redis.PubSub
	subscriptionsMu sync.Mutex
	instanceID      string
	// Maps CallID -> List of connected Clients
	// Using a simple map for now; for horizontal scaling, use Redis Pub/Sub
	calls map[string][]*Client
	mu    sync.RWMutex
}

func NewSignalingService(webrtcService *WebRTCService, allowedOrigins []string, redisClient *redis.Client) *SignalingService {
	service := &SignalingService{
		logger:        logger.NewLogger("SignalingService", logger.INFO),
		webrtcService: webrtcService,
		redisClient:   redisClient,
		subscriptions: make(map[string]*redis.PubSub),
		instanceID:    uuid.NewString(),
		calls:         make(map[string][]*Client),
	}
	service.setAllowedOrigins(allowedOrigins)
	service.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     service.checkOrigin,
	}
	return service
}

func (s *SignalingService) setAllowedOrigins(allowedOrigins []string) {
	s.allowAllOrigins = false
	s.allowedOrigins = make(map[string]struct{})
	for _, origin := range allowedOrigins {
		item := strings.TrimSpace(origin)
		if item == "" {
			continue
		}
		if item == "*" {
			s.allowAllOrigins = true
			continue
		}
		s.allowedOrigins[item] = struct{}{}
	}
}

func (s *SignalingService) checkOrigin(r *http.Request) bool {
	if s.allowAllOrigins {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	if len(s.allowedOrigins) == 0 {
		return sameHostOrigin(origin, r.Host)
	}

	_, ok := s.allowedOrigins[origin]
	return ok
}

func sameHostOrigin(origin string, host string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Host == host
}

// HandleWebSocket upgrades the connection and starts the message loop
func (s *SignalingService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. Upgrade HTTP to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
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
		_ = conn.Close()
		return
	}

	// 3. Validate Token
	tokenData, err := s.webrtcService.ValidateToken(initMsg.Token)
	if err != nil {
		s.logger.Info("Invalid signaling token: %v", err)
		if writeErr := conn.WriteJSON(map[string]string{"error": "unauthorized"}); writeErr != nil {
			s.logger.Info("Failed to send unauthorized response: %v", writeErr)
		}
		_ = conn.Close()
		return
	}

	callID := tokenData.CallID

	// 3b. Re-verify call membership against current DB state: the token may
	// still be cryptographically valid but stale (e.g. the call ended, or
	// was never this user's to join) relative to when it was issued.
	if err := s.webrtcService.VerifyMembership(callID, tokenData.UserID); err != nil {
		s.logger.Info("Signaling membership check failed: callID=%s, userID=%d, err=%v", callID, tokenData.UserID, err)
		if writeErr := conn.WriteJSON(map[string]string{"error": "unauthorized"}); writeErr != nil {
			s.logger.Info("Failed to send unauthorized response: %v", writeErr)
		}
		_ = conn.Close()
		return
	}

	client := &Client{
		conn:   conn,
		userID: tokenData.UserID,
		callID: callID,
	}

	// 4. Register Client (capped at maxPeersPerCall so a leaked/replayed
	// token can't add a third participant to the room)
	if !s.addClient(callID, client) {
		s.logger.Info("Rejected peer for call %s: room is full", callID)
		if writeErr := conn.WriteJSON(map[string]string{"error": "call room is full"}); writeErr != nil {
			s.logger.Info("Failed to send room-full response: %v", writeErr)
		}
		_ = conn.Close()
		return
	}
	s.logger.Info("Peer connected to call %s: userID=%d", callID, tokenData.UserID)
	s.ensureSubscription(callID)

	// 5. Message Loop
	defer func() {
		s.removeClient(callID, client)
		_ = conn.Close()
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
		s.publishToCall(callID, msg)
	}
}

// addClient registers client under callID and reports whether it was
// admitted. It's rejected once the call's room already holds
// maxPeersPerCall peers.
func (s *SignalingService) addClient(callID string, client *Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls[callID]) >= maxPeersPerCall {
		return false
	}
	s.calls[callID] = append(s.calls[callID], client)
	return true
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
		s.closeSubscription(callID)
	}
}

// broadcastToCall sends the message to OTHER peers in the same call
func (s *SignalingService) broadcastToCall(callID string, sender *Client, msg SignalingMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers := s.calls[callID]
	for _, peer := range peers {
		if sender == nil || peer != sender {
			if err := peer.conn.WriteJSON(msg); err != nil {
				s.logger.Info("Failed to send message to peer: %v", err)
				// We don't remove here; let the Read loop handle disconnection errors
			}
		}
	}
}

type signalingEnvelope struct {
	InstanceID string           `json:"instance_id"`
	Message    SignalingMessage `json:"message"`
}

func (s *SignalingService) channelName(callID string) string {
	return "signaling:" + callID
}

func (s *SignalingService) ensureSubscription(callID string) {
	if s.redisClient == nil {
		return
	}

	s.subscriptionsMu.Lock()
	if _, ok := s.subscriptions[callID]; ok {
		s.subscriptionsMu.Unlock()
		return
	}
	pubsub := s.redisClient.Subscribe(context.Background(), s.channelName(callID))
	s.subscriptions[callID] = pubsub
	s.subscriptionsMu.Unlock()

	go s.forwardSubscription(callID, pubsub)
}

func (s *SignalingService) closeSubscription(callID string) {
	if s.redisClient == nil {
		return
	}

	s.subscriptionsMu.Lock()
	pubsub, ok := s.subscriptions[callID]
	if ok {
		delete(s.subscriptions, callID)
	}
	s.subscriptionsMu.Unlock()

	if ok {
		_ = pubsub.Close()
	}
}

func (s *SignalingService) forwardSubscription(callID string, pubsub *redis.PubSub) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, err := pubsub.Receive(ctx)
	cancel()
	if err != nil {
		s.logger.Info("Failed to subscribe to redis signaling channel: %v", err)
		s.closeSubscription(callID)
		return
	}

	ch := pubsub.Channel()
	for msg := range ch {
		var envelope signalingEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
			s.logger.Info("Failed to decode signaling message: %v", err)
			continue
		}
		if envelope.InstanceID == s.instanceID {
			continue
		}
		s.broadcastToCall(callID, nil, envelope.Message)
	}
}

func (s *SignalingService) publishToCall(callID string, msg SignalingMessage) {
	if s.redisClient == nil {
		return
	}

	envelope := signalingEnvelope{
		InstanceID: s.instanceID,
		Message:    msg,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		s.logger.Info("Failed to encode signaling message: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.redisClient.Publish(ctx, s.channelName(callID), data).Err(); err != nil {
		s.logger.Info("Failed to publish signaling message: %v", err)
	}
}
