package network

import (
	"encoding/json"
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

// Response struct is a response that is sent back to a client from the server.
// it will contain the type of response, the status code as http code,
// any accompanying message about the status if applicable, and the data if applicable
type Response struct {
	MessageId string `json:"id"`
	Type      string `json:"type"`
	Code      int    `json:"code"`              // 200, 400, etc.
	Message   string `json:"message,omitempty"` // "OK" or error message
	Data      any    `json:"data,omitempty"`    // optional payload (topic info, schema, etc.)
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
