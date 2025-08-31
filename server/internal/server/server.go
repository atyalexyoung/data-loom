package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketMessage contains a message that is sent from the client to the server.
// Contains the action to preform, the topic to preform the action on (if applicable),
// and any accompanying data (if applicable)
type WebSocketMessage struct {
	Id         string          `json:"id"`
	Action     string          `json:"action"`
	Topic      string          `json:"topic,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	RequireAck bool            `json:"requireAck,omitempty"`
}

// Response struct is a response that is sent back to a client from the server.
// it will contain the type of response, the status code as http code,
// any accompanying message about the status if applicable, and the data if applicable
type Response struct {
	Id      string      `json:"id"`
	Type    string      `json:"type"`
	Code    int         `json:"code"`              // 200, 400, etc.
	Message string      `json:"message,omitempty"` // "OK" or error message
	Data    interface{} `json:"data,omitempty"`    // optional payload (topic info, schema, etc.)
}

// HandlerFunc is a function signature definition for all handlers of client requests.
type HandlerFunc func(*network.Client, WebSocketMessage)

// WebSocketServer is the server struct that contains the hub of clients,
// the topic manager, the websocket upgrader, and a map of all the handlers
// for the various actions that clients can take.
type WebSocketServer struct {
	hub          *network.ClientHub
	topicManager *network.TopicManager
	upgrader     websocket.Upgrader
	handlers     map[string]HandlerFunc
	config       Config
}

type Config struct {
	APIKey string
}

func loadConfig() *Config {
	cfg := &Config{}

	// Production: read from environment
	if envKey := os.Getenv("MY_SERVER_KEY"); envKey != "" {
		cfg.APIKey = envKey
	} else {
		// Development fallback
		cfg.APIKey = "data-loom-api-key"
	}

	return cfg
}

// NewWebSocketServer will create and set up a WebSocketServer struct that is ready to use.
func NewWebSocketServer(hub *network.ClientHub, topicManager *network.TopicManager) *WebSocketServer {
	s := &WebSocketServer{
		hub:          hub,
		topicManager: topicManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	loadConfig()

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

	/*
		FUTURE HANDLERS
		s.registerHandler("list", s.list, s.requireTopic)
		s.registerHandler("publishMan", s.getHandler, s.requireTopic)
		s.registerHandler("sendWithoutSave", s.registerTopicHandler, s.requireTopic, s.requireData)
		s.registerHandler("deleteManyTopics", s.unregisterTopicHandler, s.requireTopic)
		s.registerHandler("listWithPattern", s.unregisterTopicHandler, s.requireTopic)
	*/

	log.Trace("Returning new web socket server.")
	return s
}

func (s *WebSocketServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	return mux
}

// SendToClient wraps the SendJSON with error handling for websocket errors
func (s *WebSocketServer) SendToClient(c *network.Client, message any) {
	if err := c.SendJSON(message); err != nil {
		if !s.handleWebSocketError(err, c) {
			// disconnect from this loser
		} else { // we good ig, just watch it

		}
	}
}

func (s *WebSocketServer) authToken(token string) bool {
	return token == s.config.APIKey
}

// handleWebSocket is the main websocket handler that will loop to read incoming
// data from a client. This is a goroutine under the hood as handled by gorilla/websocket
// and each client will get their own handleWebSocket handler.
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {

	// TODO: authenticate client
	// --- AUTH STEP BEFORE UPGRADE ---
	// generate a new UUID for this connection
	clientID := uuid.NewString()

	// get token for auth
	token := r.Header.Get("Authorization")
	if !s.authToken(token) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// set the client ID in an HttpOnly cookie
	responseHeader := http.Header{}
	responseHeader.Set("Set-Cookie", fmt.Sprintf("client_id=%s; HttpOnly", clientID))

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
		var msg WebSocketMessage
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

// RouteMessage will take the action from a WebSocketMessage and determine which handler should take care of the logic.
func (s *WebSocketServer) RouteMessage(client *network.Client, msg WebSocketMessage) {
	log.Debugf("Routing incoming message from client: %s for action: %s", client.Id, msg.Action)
	if handler, ok := s.handlers[msg.Action]; ok {
		handler(client, msg)
	} else {
		log.Warn("Unknown action: ", msg.Action)
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
		s.SendToClient(client, Response{
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

		s.SendToClient(client, Response{
			Id:   "UNKNOWN",
			Type: "UNKOWN",
			Code: http.StatusBadRequest,
		})
		return true
	}

	ctx.Error("WebSocket read error: ", err)
	return true
}
