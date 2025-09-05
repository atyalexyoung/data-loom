package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/topic"
)

type mockTopicManager struct {
	IsMethodCalled bool
	ErrorResult    error
	ClientsResult  []*network.Client
	ClientResult   *network.Client
	BytesResult    []byte
	TopicResult    *topic.Topic
	TopicsResult   []*topic.Topic
	BoolResult     bool
}

func (tm *mockTopicManager) Subscribe(topicName string, client *network.Client) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) Unsubscribe(topicName string, client *network.Client) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) ListSubscribersForTopic(topicName string) ([]*network.Client, error) {
	tm.IsMethodCalled = true

	return tm.ClientsResult, tm.ErrorResult
}

func (tm *mockTopicManager) UnsubscribeAll(client *network.Client) {
	tm.IsMethodCalled = true
}

func (tm *mockTopicManager) Publish(topicName string, sender *network.Client, value []byte) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) SendWithoutSave(topicName string, sender *network.Client, value []byte) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) Get(topicName string) ([]byte, error) {
	tm.IsMethodCalled = true
	return tm.BytesResult, tm.ErrorResult
}

func (tm *mockTopicManager) RegisterTopic(topicName string, schema map[string]any) (*topic.Topic, error) {
	tm.IsMethodCalled = true
	return tm.TopicResult, tm.ErrorResult
}

func (tm *mockTopicManager) UnregisterTopic(topicName string) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) ListTopics() ([]*topic.Topic, error) {
	tm.IsMethodCalled = true
	return tm.TopicsResult, tm.ErrorResult
}

func (tm *mockTopicManager) UpdateSchema(topicName string, schema map[string]any) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) NextFailedClient() (*network.Client, bool) {
	return tm.ClientResult, tm.BoolResult
}

type testServer struct {
	*WebSocketServer
	sent []any
}

func (s *testServer) SendToClient(c *network.Client, message any) {
	s.sent = append(s.sent, message)
}

func TestRegisterTopicHandlerSuccessWithAck(t *testing.T) {
	s := testServer{
		WebSocketServer: &WebSocketServer{
			topicManager: &mockTopicManager{},
		},
	}
	s.WebSocketServer.sender = &s

	client := &network.Client{}
	msg := network.WebSocketMessage{
		Topic:      "foo",
		RequireAck: true,
	}

	s.subscribeHandler(client, msg)

	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusOK {
		t.Error("expected status 200")
	}
}

func TestRegisterTopicHandlerSuccess(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: nil,
	}
	s := testServer{
		WebSocketServer: &WebSocketServer{
			topicManager: m,
		},
	}
	s.WebSocketServer.sender = &s

	client := &network.Client{}
	s.registerTopicHandler(client, registerTopicMessage)

	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusOK {
		t.Error("expected status 200")
	}
}

func TestRegisterHandlerFailFromTopicManager(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: fmt.Errorf("error from topic manager"),
	}
	s := testServer{
		WebSocketServer: &WebSocketServer{
			topicManager: m,
		},
	}
	s.WebSocketServer.sender = &s
	client := &network.Client{}

	s.registerTopicHandler(client, registerTopicMessage)

	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status internal server error.")
	}
}

func TestRegisterHandlerFailFromMalformedJSON(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: fmt.Errorf("error from topic manager"),
	}
	s := testServer{
		WebSocketServer: &WebSocketServer{
			topicManager: m,
		},
	}
	s.WebSocketServer.sender = &s
	client := &network.Client{}
	s.registerTopicHandler(client, registerTopicBadJSON)

	if m.IsMethodCalled {
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status internal server error.")
	}
}

var registerTopicMessage = network.WebSocketMessage{
	Id:         "registerTopicWithAck",
	Action:     "registerTopic",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"info":"example"}`),
	RequireAck: true,
}

var registerTopicBadJSON = network.WebSocketMessage{
	Id:         "registerTopicBadJSON",
	Action:     "registerTopic",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"invalid":}`), // malformed JSON
	RequireAck: true,
}

var subscribeWithAck = network.WebSocketMessage{
	Id:         "subscribeWithAck",
	Action:     "subscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"filter":"none"}`),
	RequireAck: true,
}

var subscribeWithoutAck = network.WebSocketMessage{
	Id:         "subscribeWithoutAck",
	Action:     "subscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"filter":"none"}`),
	RequireAck: false,
}

var unsubscribeWithAck = network.WebSocketMessage{
	Id:         "unsubscribeWithAck",
	Action:     "unsubscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{}`),
	RequireAck: true,
}

var unsubscribeWithoutAck = network.WebSocketMessage{
	Id:         "unsubscribeWithoutAck",
	Action:     "unsubscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{}`),
	RequireAck: false,
}

var publishWithAck = network.WebSocketMessage{
	Id:         "publishWithAck",
	Action:     "publish",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	RequireAck: true,
}

var publishWithoutAck = network.WebSocketMessage{
	Id:         "publishWithoutAck",
	Action:     "publish",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	RequireAck: false,
}
