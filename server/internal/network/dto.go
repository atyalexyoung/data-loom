package network

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

// WebSocketMessage contains a message that is sent from the client to the server.
// Contains the action to preform, the topic to preform the action on (if applicable),
// and any accompanying data (if applicable)
type WebSocketMessage struct {
	MessageId  string          `json:"id"`
	SenderId   string          `json:"senderId,omitempty"`
	Action     string          `json:"action"`
	Topic      string          `json:"topic,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	RequireAck bool            `json:"requireAck,omitempty"`
	ParsedData map[string]any  `json:"-"`
}

func (msg *WebSocketMessage) GetLogFields() log.Fields {
	return log.Fields{
		"MessageId":  msg.MessageId,
		"SenderId":   msg.SenderId,
		"Action":     msg.Action,
		"Topic":      msg.Topic,
		"Data":       msg.Data,
		"RequireAck": msg.RequireAck,
		"ParsedData": msg.ParsedData,
	}
}

// Response struct is a response that is sent back to a client from the server.
// it will contain the type of response, the status code as http code,
// any accompanying message about the status if applicable, and the data if applicable
type Response struct {
	MessageId string `json:"id"`
	Action    string `json:"action"`
	Code      int    `json:"code"`              // 200, 400, etc.
	Message   string `json:"message,omitempty"` // "OK" or error message
	Data      any    `json:"data,omitempty"`    // optional payload (topic info, schema, etc.)
	Type      string `json:"type,omitempty"`    // "response" for clients to tell if something is response or request.
}

func (response *Response) GetLogFields() log.Fields {
	return log.Fields{
		"MessageId": response.MessageId,
		"Action":    response.Action,
		"Code":      response.Code,
		"Message":   response.Message,
		"Data":      response.Data,
		"Type":      response.Type,
	}
}

func NewResponse(msg WebSocketMessage, code int, message string, data any) Response {
	return Response{
		MessageId: msg.MessageId,
		Action:    msg.Action,
		Message:   message,
		Code:      code,
		Data:      data,
		Type:      "response",
	}
}

// TopicSchemaResponse will contain information a client would want to
// know about a topic schema
type TopicSchemaResponse struct {
	Version int            `json:"version"`
	Schema  map[string]any `json:"schema"`
}

// TopicResponse is the struct that will contain the information a client
// would want to know about a topic
type TopicResponse struct {
	Name   string              `json:"name"`
	Schema TopicSchemaResponse `json:"schema"`
}
