package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Action string          `json:"action"`
	Key    string          `json:"key,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type HandlerFunc func(*network.Client, WebSocketMessage)
type WebSocketServer struct {
	hub          *network.ClientHub
	topicManager *network.TopicManager
	upgrader     websocket.Upgrader
	handlers     map[string]HandlerFunc
}

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

	log.Println("Setting up handlers...")
	s.handlers = map[string]HandlerFunc{
		"subscribe": func(c *network.Client, msg WebSocketMessage) {
			s.topicManager.Subscribe(msg.Key, c)
		},
		"publish": func(c *network.Client, msg WebSocketMessage) {
			s.topicManager.Publish(msg.Key, c, msg.Data)
		},
		"unsubscribe": func(c *network.Client, msg WebSocketMessage) {
			s.topicManager.Unsubscribe(msg.Key, c)
		},
		"unsubscribeAll": func(c *network.Client, msg WebSocketMessage) {
			s.topicManager.UnsubscribeAll(c)
		},
		/*
			/* FUTURE HANDLERS
			"get": s.TopicManager.GetValue(),
			"list": s.TopicManager.ListTopics(),
			"registerTopic": s.TopicManager.RegisterTopic(),
			"deleteTopic": s.TopicManager.DeleteTopic(),
			"publishMany": s.TopicManager.SetManyTopics(),
			"sendWithoutSave": s.TopicManager.SendWithoutSave(),
			"deleteManyTopics": s.TopicManager.DeleteManyTopics(),
			"listWithPattern": s.TopicManager.ListWithPattern(),
		*/
	}

	return s
}

func (s *WebSocketServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	return mux
}

// go routine that will handle a client connecting
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error", err)
		return
	}
	defer conn.Close()

	client := &network.Client{Conn: conn, Id: uuid.NewString()}

	// TODO: authenticate user

	s.hub.AddClient(client)
	defer s.hub.RemoveClient(client)

	for {
		var msg WebSocketMessage
		// blocks until can read message
		if err := conn.ReadJSON(&msg); err != nil {
			// Check if it's a close error
			var ce *websocket.CloseError
			if errors.As(err, &ce) {
				switch ce.Code {
				case websocket.CloseNormalClosure:
					log.Println("Normal closure")
				case websocket.CloseGoingAway:
					log.Println("Client going away")
				case websocket.CloseAbnormalClosure:
					log.Println("Abnormal closure")
				default:
					log.Printf("Other close code %d: %v", ce.Code, ce.Text)
				}

				// if the connection is closed, get this guy outta here
				s.topicManager.UnsubscribeAll(client)
				break
			}

			// Check if it's a JSON syntax/unmarshal error
			var syntaxErr *json.SyntaxError
			if errors.As(err, &syntaxErr) {
				log.Printf("JSON syntax error at byte %d: %v", syntaxErr.Offset, syntaxErr.Error())

				// TODO: send back to the user their json sucks
				// just continue to drop this message and not disconnect
				continue
			}

			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &typeErr) {
				log.Printf("JSON type error: expected %v but got %v at byte %d", typeErr.Type, typeErr.Value, typeErr.Offset)
				// just continue to drop this message and not disconnect
				continue
			}

			// Fallback for any other error
			log.Printf("Read error: %v", err)
			break
		}

		s.RouteMessage(client, msg)
	}
}

// RouteMessage will take the action from a WebSocketMessage and determine which handler should take care of the logic.
func (s *WebSocketServer) RouteMessage(client *network.Client, msg WebSocketMessage) {
	if handler, ok := s.handlers[msg.Action]; ok {
		handler(client, msg)
	} else {
		log.Println("Unknown action: ", msg.Action)
	}
}
