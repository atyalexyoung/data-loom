package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/topic"
)

// ----------------------------------------------------------------------- mock topic manager

type mockTopicManager struct {
	IsMethodCalled bool
	ErrorResult    error
	ClientsResult  []*network.Client
	ClientResult   *network.Client
	BytesResult    []byte
	TopicResult    *topic.Topic
	TopicsResult   []*topic.Topic
	BoolResult     bool
	MapResult      map[string]any
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

func (tm *mockTopicManager) Publish(ctx context.Context, msg network.WebSocketMessage, sender *network.Client, value map[string]any, errCh chan error) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) SendWithoutSave(ctx context.Context, msg network.WebSocketMessage, sender *network.Client, value map[string]any, errCh chan error) error {
	tm.IsMethodCalled = true
	return tm.ErrorResult
}

func (tm *mockTopicManager) Get(ctx context.Context, topicName string) (map[string]any, error) {
	tm.IsMethodCalled = true
	return tm.MapResult, tm.ErrorResult
}

func (tm *mockTopicManager) RegisterTopic(topicName string, schema map[string]any) (*topic.Topic, error) {
	tm.IsMethodCalled = true
	return tm.TopicResult, tm.ErrorResult
}

func (tm *mockTopicManager) UnregisterTopic(ctx context.Context, topicName string) error {
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

func (tm *mockTopicManager) IsSchemaMatch(topicName string, schema map[string]any) (bool, error) {
	return tm.BoolResult, tm.ErrorResult
}

//------------------------------------------------------------------------------ test server

type testServer struct {
	*WebSocketServer
	sent []any
}

func (s *testServer) SendToClient(c *network.Client, message any) {
	s.sent = append(s.sent, message)
}

func SetupStuff(m *mockTopicManager) (*testServer, *network.Client) {
	s := testServer{
		WebSocketServer: &WebSocketServer{
			topicManager: m,
		},
	}
	s.WebSocketServer.sender = &s
	client := &network.Client{}
	return &s, client
}

//------------------------------------------------------------------- subscribe handler tests

var subscribeWithAck = network.WebSocketMessage{
	Id:         "subscribeWithAck",
	Action:     "subscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"filter":"none"}`),
	ParsedData: map[string]any{"filter": "none"},
	RequireAck: true,
}

var subscribeWithoutAck = network.WebSocketMessage{
	Id:         "subscribeWithoutAck",
	Action:     "subscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"filter":"none"}`),
	ParsedData: map[string]any{"filter": "none"},
	RequireAck: false,
}

func TestSubscribeHandlerSuccessWithAck(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: nil,
	}
	s, c := SetupStuff(m)

	s.subscribeHandler(c, subscribeWithAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	fmt.Println(len(s.sent))
	if len(s.sent) != 1 { // if nothing was sent, bad. We have ack
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusOK { // if response wasn't ok, bad
		t.Error("expected status internal server error.")
	}
}

func TestSubscribeHandlerSuccessWithoutAck(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: nil,
	}
	s, c := SetupStuff(m)

	s.subscribeHandler(c, subscribeWithoutAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 0 { // if nothing was sent, good. We have ack disabled
		t.Fatal("expected no message with success and no ack")
		// do nothing since this is expected
	}
}

func TestSubscribeHandlerFailFromTopicManager(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: fmt.Errorf("error from topic manager"),
	}
	s, c := SetupStuff(m)

	s.subscribeHandler(c, subscribeWithoutAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 1 { // if nothing was sent, bad. We have ack
		t.Fatal("expected 1 message sent to client.")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status internal server error.")
	}
}

//------------------------------------------------------------------ unsubscribe handler tests

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

func TestUnsubscribeHandlerSuccessWithAck(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: nil,
	}
	s, c := SetupStuff(m)

	s.unsubscribeHandler(c, unsubscribeWithAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	fmt.Println(len(s.sent))
	if len(s.sent) != 1 { // if nothing was sent, bad. We have ack
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusOK { // if response wasn't ok, bad
		t.Error("expected status internal server error.")
	}
}

func TestUnsubscribeHandlerSuccessWithoutAck(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: nil,
	}
	s, c := SetupStuff(m)

	s.unsubscribeHandler(c, unsubscribeWithoutAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 0 { // if nothing was sent, good. We have ack disabled
		t.Fatal("expected no message with success and no ack")
		// do nothing since this is expected
	}
}

func TestUnsubscribeHandlerFailFromTopicManager(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: fmt.Errorf("error from topic manager"),
	}
	s, c := SetupStuff(m)

	s.unsubscribeHandler(c, unsubscribeWithAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 1 { // if nothing was sent, bad. We have ack
		t.Fatal("expected 1 message sent to client.")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status internal server error.")
	}
}

//------------------------------------------------------------------- publish handler tests

var publishSuccessWithAck = network.WebSocketMessage{
	Id:         "publishWithAck",
	Action:     "publish",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: true,
}

var publishSuccessWithoutAck = network.WebSocketMessage{
	Id:         "publishWithoutAck",
	Action:     "publish",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: false,
}

var publishFailFromTopicManager = network.WebSocketMessage{
	Id:         "publishWithoutAck",
	Action:     "publish",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: false,
}

var publishFailFromNoDataSupplied = network.WebSocketMessage{
	Id:     "publishWithoutAck",
	Action: "publish",
	Topic:  "testTopic",
	Data:   json.RawMessage(`{"message":"hello world"}`),
	//ParsedData: none
	RequireAck: false,
}

func TestPublishSuccessWithAck(t *testing.T) {
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
	s.publishHandler(client, publishSuccessWithAck)

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

func TestPublishSuccessWithoutAck(t *testing.T) {
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
	s.publishHandler(client, publishSuccessWithoutAck)

	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 0 {
		t.Fatal("expected no message")
	}
}

func TestPublishFailFromTopicManager(t *testing.T) {
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
	s.publishHandler(client, publishFailFromTopicManager)

	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status 200")
	}
}

func TestPublishFailFromNoParsedData(t *testing.T) {
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
	s.publishHandler(client, publishFailFromNoDataSupplied)

	if m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusBadRequest {
		t.Error("expected status 200")
	}
}

//----------------------------------------------------------------------- get handler tests

//------------------------------------------------------------------- register handler tests

var registerTopicSuccesssMsg = network.WebSocketMessage{
	Id:         "registerTopicWithAck",
	Action:     "registerTopic",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"info":"example"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: true,
}

var registerTopicFailNoParsedData = network.WebSocketMessage{
	Id:     "registerTopicBadJSON",
	Action: "registerTopic",
	Topic:  "testTopic",
	Data:   json.RawMessage(`{"invalid":}`),
	// ParsedData: none
	RequireAck: true,
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
	s.registerTopicHandler(client, registerTopicSuccesssMsg)

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

	s.registerTopicHandler(client, registerTopicSuccesssMsg)

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

func TestRegisterHandlerFailFromNoData(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: fmt.Errorf("error from topic manager"),
	}

	s, client := SetupStuff(m)

	s.registerTopicHandler(client, registerTopicFailNoParsedData)

	if m.IsMethodCalled {
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusBadRequest {
		t.Error("expected status internal server error.")
	}
}

//------------------------------------------------------------------ unregister handler tests

var unregisterWithAck = network.WebSocketMessage{
	Id:         "unsubscribeWithAck",
	Action:     "unsubscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{}`),
	RequireAck: true,
}

var unregisterWithoutAck = network.WebSocketMessage{
	Id:         "unsubscribeWithoutAck",
	Action:     "unsubscribe",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{}`),
	RequireAck: false,
}

func TestUnregisterHandlerSuccessWithAck(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: nil,
	}
	s, c := SetupStuff(m)

	s.unregisterTopicHandler(c, unregisterWithAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	fmt.Println(len(s.sent))
	if len(s.sent) != 1 { // if nothing was sent, bad. We have ack
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusOK { // if response wasn't ok, bad
		t.Error("expected status internal server error.")
	}
}

func TestUnregisterHandlerSuccessWithoutAck(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: nil,
	}
	s, c := SetupStuff(m)

	s.unsubscribeHandler(c, unregisterWithoutAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 0 { // if nothing was sent, good. We have ack disabled
		t.Fatal("expected no message with success and no ack")
		// do nothing since this is expected
	}
}

func TestUnregisterHandlerFailFromTopicManager(t *testing.T) {
	m := &mockTopicManager{
		ErrorResult: fmt.Errorf("error from topic manager"),
	}
	s, c := SetupStuff(m)

	s.unregisterTopicHandler(c, unregisterWithoutAck)

	if !m.IsMethodCalled { // if method wasn't called, bad
		t.Error("expected topic manager method to not be called but was.")
	}
	if len(s.sent) != 1 { // if nothing was sent, bad. We have ack
		t.Fatal("expected 1 message sent to client.")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status internal server error.")
	}
}

//----------------------------------------------------------------- list topics handler tests

//-------------------------------------------------------------- update schema  handler tests

var updateSchemaSuccessWithAck = network.WebSocketMessage{
	Id:         "sendWithoutSaveSuccessWithAck",
	Action:     "sendWithoutSave",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: true,
}

var updateSchemaSuccessWithoutAck = network.WebSocketMessage{
	Id:         "sendWithoutSaveSuccessWithoutAck",
	Action:     "sendWithoutSave",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: false,
}

var updateSchemaFailFromNoDataSupplied = network.WebSocketMessage{
	Id:     "sendWithoutSaveFailFromNoDataSupplied",
	Action: "sendWithoutSave",
	Topic:  "testTopic",
	Data:   json.RawMessage(`{"message":"hello world"}`),
	//ParsedData: none
	RequireAck: false,
}

func TestUpdateSchemaSuccessWithAck(t *testing.T) {
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
	s.updateSchemaHandler(client, updateSchemaSuccessWithAck)

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

func TestUpdateSchemaSuccessWithoutAck(t *testing.T) {
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
	s.updateSchemaHandler(client, updateSchemaSuccessWithoutAck)
	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 0 {
		t.Fatal("expected no message")
	}
}

func TestUpdateSchemaFailFromTopicManager(t *testing.T) {
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
	s.updateSchemaHandler(client, updateSchemaSuccessWithAck)

	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status 200")
	}
}

func TestUpdateSchemaFailFromNoParsedData(t *testing.T) {
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
	s.updateSchemaHandler(client, updateSchemaFailFromNoDataSupplied)

	if m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusBadRequest {
		t.Error("expected status 200")
	}
}

//----------------------------------------------------------- send without save handler tests

var sendWithoutSaveSuccessWithAck = network.WebSocketMessage{
	Id:         "sendWithoutSaveSuccessWithAck",
	Action:     "sendWithoutSave",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: true,
}

var sendWithoutSaveSuccessWithoutAck = network.WebSocketMessage{
	Id:         "sendWithoutSaveSuccessWithoutAck",
	Action:     "sendWithoutSave",
	Topic:      "testTopic",
	Data:       json.RawMessage(`{"message":"hello world"}`),
	ParsedData: map[string]any{"message": "hello world"},
	RequireAck: false,
}

var sendWithoutSaveFailFromNoDataSupplied = network.WebSocketMessage{
	Id:     "sendWithoutSaveFailFromNoDataSupplied",
	Action: "sendWithoutSave",
	Topic:  "testTopic",
	Data:   json.RawMessage(`{"message":"hello world"}`),
	//ParsedData: none
	RequireAck: false,
}

func TestSendWithoutSaveSuccessWithAck(t *testing.T) {
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
	s.sendWithoutSaveHandler(client, sendWithoutSaveSuccessWithAck)

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

func TestSendWithoutSaveSuccessWithoutAck(t *testing.T) {
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
	s.sendWithoutSaveHandler(client, sendWithoutSaveSuccessWithoutAck)

	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 0 {
		t.Fatal("expected no message")
	}
}

func TestSendWithoutSaveFailFromTopicManager(t *testing.T) {
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
	s.sendWithoutSaveHandler(client, sendWithoutSaveSuccessWithAck)

	if !m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusInternalServerError {
		t.Error("expected status 200")
	}
}

func TestSendWithoutSaveFailFromNoParsedData(t *testing.T) {
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
	s.sendWithoutSaveHandler(client, sendWithoutSaveFailFromNoDataSupplied)

	if m.IsMethodCalled {
		t.Error("expected topic manager method to be called but wasn't.")
	}
	if len(s.sent) != 1 {
		t.Fatal("expected 1 message")
	}
	resp, ok := s.sent[0].(network.Response)
	if !ok || resp.Code != http.StatusBadRequest {
		t.Error("expected status 200")
	}
}
