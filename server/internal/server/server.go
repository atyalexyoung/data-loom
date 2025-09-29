package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/config"
	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/topic"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	FAILED_MESSAGE_THRESHOLD = 3
)

type MessageSender interface {
	SendToClient(c *network.Client, message any)
}

// HandlerFunc is a function signature definition for all handlers of client requests.
type HandlerFunc func(*network.Client, network.WebSocketMessage)

// WebSocketServer is the server struct that contains the hub of clients,
// the topic manager, the websocket upgrader, and a map of all the handlers
// for the various actions that clients can take.
type WebSocketServer struct {
	sender        MessageSender
	hub           *network.ClientHub
	topicManager  topic.TopicManager
	upgrader      websocket.Upgrader
	handlers      map[string]HandlerFunc
	config        *config.Config
	failedClients map[*network.Client]int
	mu            sync.RWMutex
}

// NewWebSocketServer will create and set up a WebSocketServer struct that is ready to use.
func NewWebSocketServer(hub *network.ClientHub, topicManager topic.TopicManager, config *config.Config) *WebSocketServer {
	s := &WebSocketServer{
		hub:          hub,
		topicManager: topicManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		handlers: make(map[string]HandlerFunc),
		config:   config,
	}
	s.sender = s

	// these handlers are set up with decorators for "middleware-like" functionality by
	// wrapping the the inner-most handler with decorators for pre/post hooks for things
	// like metrics, logging, validation or auth with early returns to block handler etc.

	log.Debug("Setting up handlers...")
	s.registerHandler("subscribe", s.subscribeHandler, s.metricsDecorator, s.requireTopicDecorator)
	s.registerHandler("publish", s.publishHandler, s.metricsDecorator, s.requireDataDecorator, s.requireTopicDecorator)
	s.registerHandler("unsubscribe", s.unsubscribeHandler, s.metricsDecorator, s.requireTopicDecorator)
	s.registerHandler("unsubscribeAll", s.unsubscribeAllHandler, s.metricsDecorator, s.requireTopicDecorator)
	s.registerHandler("get", s.getHandler, s.metricsDecorator, s.requireTopicDecorator)
	s.registerHandler("registerTopic", s.registerTopicHandler, s.metricsDecorator, s.requireDataDecorator, s.requireTopicDecorator)
	s.registerHandler("unregisterTopic", s.unregisterTopicHandler, s.metricsDecorator, s.requireTopicDecorator)
	s.registerHandler("listTopics", s.listTopicsHandler, s.metricsDecorator) // no required topics
	s.registerHandler("updateSchema", s.updateSchemaHandler, s.metricsDecorator, s.requireTopicDecorator, s.requireDataDecorator)
	s.registerHandler("sendWithoutSave", s.sendWithoutSaveHandler, s.metricsDecorator, s.requireTopicDecorator, s.requireDataDecorator)

	/*
		FUTURE HANDLERS
		s.registerHandler("publishMany", s.getHandler, s.requireTopic)
		s.registerHandler("sendWithoutSave", s.registerTopicHandler, s.requireTopic, s.requireData)
		s.registerHandler("deleteManyTopics", s.unregisterTopicHandler, s.requireTopic)
		s.registerHandler("listWithPattern", s.unregisterTopicHandler, s.requireTopic)
	*/

	log.Trace("Returning new web socket server.")
	return s
}

// Handler returns ta handler for the websocket connection to main
func (s *WebSocketServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	return mux
}

// SendToClient wraps the SendJSON with error handling for websocket errors
func (s *WebSocketServer) SendToClient(c *network.Client, message any) {
	if err := c.SendJSON(message); err != nil {
		if !s.handleWebSocketError(err, c) {
			s.MarkClientFailed(c)
		} else { // we good ig, just watch it

		}
	}
}

// handleWebSocket is the main websocket handler that will loop to read incoming
// data from a client. This is a goroutine under the hood as handled by gorilla/websocket
// and each client will get their own handleWebSocket handler.
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {

	if s.config.APIKey != "" {
		apiKey := strings.TrimSpace(r.Header.Get("Authorization"))
		if apiKey != s.config.APIKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	} // else the API key not required.

	clientID := r.Header.Get("ClientId")
	if clientID == "" {
		clientID = uuid.NewString() // fallback to generated ID
	}

	// check if the client Id is already used or not.
	if currentClient := s.hub.GetClient(clientID); currentClient != nil {
		http.Error(w, "client ID already exists", http.StatusConflict)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"request": r,
		}).Errorf("Upgrade error: %v", err)
		return
	}
	defer conn.Close()
	client := &network.Client{Conn: conn, Id: uuid.NewString()}

	// send back the uuid of client

	s.hub.AddClient(client)
	defer s.hub.RemoveClient(client)

	for {
		var msg network.WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil { // blocks until can read message
			if !s.handleWebSocketError(err, client) { // returns bool if client is ok
				// if we aren't ok, disconnect from this loser
				break
			}
		} else { // we all good
			s.RouteMessage(client, msg)
		}
	}
}

// ListenForClientFailuresFromTopicManager will get clients that have
// failed from the topic manager to be marked as failed by the server
func (s *WebSocketServer) ListenForClientFailuresFromTopicManager() {
	for {
		client, ok := s.topicManager.NextFailedClient()
		if !ok {
			// channel was closed, close loop
			return
		}
		s.MarkClientFailed(client)
	}
}

// MarkClientFailed will increment the client's failures.
func (s *WebSocketServer) MarkClientFailed(c *network.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failedClients[c]++
}

// StartClientCleanupCrew will start a goroutine that will periodically cleanup clients failing to communicate.
func (s *WebSocketServer) StartClientCleanupCrew(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupFailedClients()
			}
		}
	}()
}

// cleanupFailedClients will remove the clients that are failing to communicate.
func (s *WebSocketServer) cleanupFailedClients() {
	s.mu.Lock()
	defer s.mu.Unlock()

	removals := make([]*network.Client, 0)

	for client, numFails := range s.failedClients {
		if numFails > FAILED_MESSAGE_THRESHOLD {
			s.topicManager.UnsubscribeAll(client)
			s.hub.RemoveClient(client)
			removals = append(removals, client)
		}
	}

	for _, client := range removals {
		delete(s.failedClients, client)
	}
}

// RouteMessage will take the action from a WebSocketMessage and determine which handler should take care of the logic.
func (s *WebSocketServer) RouteMessage(client *network.Client, msg network.WebSocketMessage) {
	log.Debugf("Routing incoming message from client: %s for action: %s", client.Id, msg.Action)
	if handler, ok := s.handlers[msg.Action]; ok {
		handler(client, msg)
	} else {
		log.Warn("Unknown action: ", msg.Action)
		s.AckResponseBadRequest(client, msg, fmt.Errorf("unknown action: %s", msg.Action))
	}
}

// handleWebSocketError handles what to log or do with an error from the websocket.
// Returns boolean if the websocket is ok or not. If false, the client should be disconnected
func (s *WebSocketServer) handleWebSocketError(err error, client *network.Client) bool {
	ctx := log.WithField("client_id", client.Id)

	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		switch ce.Code {
		case websocket.CloseNormalClosure:
			log.Error("Normal closure")
		case websocket.CloseGoingAway:
			log.Error("Client going away")
		case websocket.CloseAbnormalClosure:
			log.Error("Abnormal closure")
		default:
			log.Errorf("Other close code %d: %v", ce.Code, ce.Text)
		}

		// if the connection is closed, get this guy outta here
		s.topicManager.UnsubscribeAll(client)
		return false
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		ctx.WithField("offset", syntaxErr.Offset).Error("JSON syntax error")
		s.sender.SendToClient(client, network.Response{
			Id:   "UNKNOWN",
			Type: "UNKOWN",
			Code: http.StatusBadRequest,
		})
		return true
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		ctx.WithFields(log.Fields{
			"expected": typeErr.Type,
			"value":    typeErr.Value,
			"offset":   typeErr.Offset,
		}).Error("JSON type error")

		s.sender.SendToClient(client, network.Response{
			Id:   "UNKNOWN",
			Type: "UNKOWN",
			Code: http.StatusBadRequest,
		})
		return true
	}

	ctx.Error("WebSocket read error: ", err)
	return true
}
