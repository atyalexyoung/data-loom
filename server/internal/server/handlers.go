package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"

	logger "github.com/atyalexyoung/data-loom/server/internal/logging"
	"github.com/atyalexyoung/data-loom/server/internal/network"
)

// parseJSON takes a type to parse JSON into, and the data of the json and
// will attempt to Unmarshal data. Returns error from unmarshaling if applicable.
func parseJSON[T any](data json.RawMessage) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}

// AckResponseSuccess with handle logging and responding to client if action was successful
func (s *WebSocketServer) AckResponseSuccess(c *network.Client, msg network.WebSocketMessage) {
	logger.HandlerSuccess(c.Id, msg.Action, msg.Topic, msg.Id)
	if msg.RequireAck {
		s.sender.SendToClient(c, network.Response{
			Id:   msg.Id,
			Type: msg.Action,
			Code: http.StatusOK,
		})
		logger.HandlerAck(c.Id, msg.Action, msg.Topic, msg.Id)
	}
}

// AckResponseData will handle logging and response with data to client.
func (s *WebSocketServer) AckResponseSuccessWithData(c *network.Client, msg network.WebSocketMessage, data any) {
	logger.HandlerSuccess(c.Id, msg.Action, msg.Topic, msg.Id)
	s.sender.SendToClient(c, network.Response{
		Id:   msg.Id,
		Type: msg.Action,
		Code: http.StatusOK,
		Data: data,
	})
	logger.HandlerAck(c.Id, msg.Action, msg.Topic, msg.Id)
}

// AckResponseError will handle logging and creating response to the client if an error has occured
func (s *WebSocketServer) AckResponseError(c *network.Client, msg network.WebSocketMessage, err error) {
	logger.HandlerError(c.Id, msg.Action, msg.Topic, msg.Id, err)
	s.sender.SendToClient(c, network.Response{
		Id:      msg.Id,
		Type:    msg.Action,
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	})
}

func (s *WebSocketServer) AckResponseBadRequest(c *network.Client, msg network.WebSocketMessage, err error) {
	logger.HandlerError(c.Id, msg.Action, msg.Topic, msg.Id, err)
	s.sender.SendToClient(c, network.Response{
		Id:      msg.Id,
		Type:    msg.Action,
		Code:    http.StatusBadRequest,
		Message: err.Error(),
	})
}

func (s *WebSocketServer) AckResponseDatabaseError(c *network.Client, msg network.WebSocketMessage, err error) {
	logger.HandlerError(c.Id, msg.Action, msg.Topic, msg.Id, err)
	s.sender.SendToClient(c, network.Response{
		Id:      msg.Id,
		Type:    "persist",
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	})
}

// Will register a handler with the action string as the lookup for the handler,
// the handler function, and any number of decorators to wrap the handler. Note: The decorators
// are ran right to left in order. In other words, the left-most decorator is the "inner-most"
// and they are wrapped around from there.
func (s *WebSocketServer) registerHandler(action string, handler HandlerFunc, decorators ...func(HandlerFunc) HandlerFunc) {
	// gets the name using reflection to log that the handler was registered
	name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	log.Tracef("Registering handler for action=%s, function=%s", action, name)

	final := handler
	for _, dec := range decorators {
		final = dec(final)
	}
	s.handlers[action] = final
}

// subscribeHandler handles subscription request, error handling from trying to subscribe
// and response to the client.
func (s *WebSocketServer) subscribeHandler(c *network.Client, msg network.WebSocketMessage) {
	if err := s.topicManager.Subscribe(msg.Topic, c); err != nil {
		s.AckResponseError(c, msg, err)
	} else {
		s.AckResponseSuccess(c, msg)
	}
}

// unsubscribeHandler handles request to unsubscribe, error handling from topic manager doing work,
// and sending response to the requesting client.
func (s *WebSocketServer) unsubscribeHandler(c *network.Client, msg network.WebSocketMessage) {
	if err := s.topicManager.Unsubscribe(msg.Topic, c); err != nil {
		s.AckResponseError(c, msg, err)
	} else {
		s.AckResponseSuccess(c, msg)
	}
}

// publishHandler handles getting the request to publish from a client, error handling
// from trying to publish, and response to the sending client.
func (s *WebSocketServer) publishHandler(c *network.Client, msg network.WebSocketMessage) {

	if msg.ParsedData == nil { // if no parsed data supplied, bad
		s.AckResponseBadRequest(c, msg, fmt.Errorf("data payload could not be parsed"))
		return
	}

	// get the current schema for this topic
	isMatch, err := s.topicManager.IsSchemaMatch(msg.Topic, msg.ParsedData)
	if err != nil || !isMatch { // if we get an error, just blame it on client for now.
		s.AckResponseBadRequest(c, msg, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		select {
		case err := <-errCh:
			if err != nil {
				s.AckResponseDatabaseError(c, msg, err)
			}
		case <-ctx.Done():
			log.WithFields(log.Fields{
				"topic":  msg.Topic,
				"client": c.Id,
			}).Warnf("DB write timeout for topic: %s, client: %s", msg.Topic, c.Id)
			s.AckResponseDatabaseError(c, msg, fmt.Errorf("timeout when persisting"))
		}
	}()

	if err := s.topicManager.Publish(ctx, msg, c, msg.ParsedData, errCh); err != nil {
		s.AckResponseError(c, msg, err)
	} else {
		s.AckResponseSuccess(c, msg)
	}
}

// unsubscribAllHandler handles the request from client to unsubscribe from all topics,
// and sending respone to the requesting client.
func (s *WebSocketServer) unsubscribeAllHandler(c *network.Client, msg network.WebSocketMessage) {
	s.topicManager.UnsubscribeAll(c)
	s.AckResponseSuccess(c, msg)
}

// getHandler handles a request to get a topic value, errors from topic manager, and
// sending response to requesting client.
func (s *WebSocketServer) getHandler(c *network.Client, msg network.WebSocketMessage) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if data, err := s.topicManager.Get(ctx, msg.Topic); err != nil {
		s.AckResponseError(c, msg, err)
	} else {
		s.AckResponseSuccessWithData(c, msg, data)
	}
}

// registerTopicHandler handles a request to register a topic, error from topic manager from
// operation, and sending response to the requesting client.
func (s *WebSocketServer) registerTopicHandler(c *network.Client, msg network.WebSocketMessage) {
	// get schema from message
	if msg.ParsedData == nil {
		s.AckResponseBadRequest(c, msg, fmt.Errorf("data payload could not be parsed"))
		return
	}

	topic, err := s.topicManager.RegisterTopic(msg.Topic, msg.ParsedData)
	if err != nil {
		s.AckResponseError(c, msg, err)
	} else if msg.RequireAck { // explicit check for requireAck since response with data doesn't
		s.AckResponseSuccessWithData(c, msg, topic)
	}
}

// unregisterTopicHandler handles request from client to unregister a topic, error from topic
// manager doing work, and responding to the requesting client.
func (s *WebSocketServer) unregisterTopicHandler(c *network.Client, msg network.WebSocketMessage) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.topicManager.UnregisterTopic(ctx, msg.Topic); err != nil {
		s.AckResponseError(c, msg, err)
	} else {
		s.AckResponseSuccess(c, msg)
	}
}

// listToipicsHandler handles request from client to get a list of users, error from topic manager
// and translating slice of topics into a list of Topic Responses from topic manager,
// and sending response to the client.
func (s *WebSocketServer) listTopicsHandler(c *network.Client, msg network.WebSocketMessage) {

	topics, err := s.topicManager.ListTopics()
	if err != nil {
		s.AckResponseError(c, msg, err)
		return
	}

	// get responses from topics
	var response []network.TopicResponse
	for _, topic := range topics {

		// get schema from topic
		var schemaResponse network.TopicSchemaResponse
		if schema, err := topic.GetLatestSchema(); err != nil {
			log.Errorf("Error when getting schema for topic: %s when getting list of topics.", topic.Name())
		} else if schema != nil {
			schemaResponse = network.TopicSchemaResponse{
				Version: schema.Version,
				Schema:  schema.Schema,
			}
		}

		response = append(response, network.TopicResponse{
			Name:   topic.Name(),
			Schema: schemaResponse,
		})
	}

	s.AckResponseSuccessWithData(c, msg, response)
}

// updateSchemaHandler handles request from client to update the schema for a topic,
// verifying parsed data, and errors from
// topic manager perfroming actions, and sending response to client.
func (s *WebSocketServer) updateSchemaHandler(c *network.Client, msg network.WebSocketMessage) {
	if msg.ParsedData == nil {
		s.AckResponseBadRequest(c, msg, fmt.Errorf("data payload could not be parsed"))
		return
	}

	err := s.topicManager.UpdateSchema(msg.Topic, msg.ParsedData)
	if err != nil {
		s.AckResponseError(c, msg, err)
		return
	}

	s.AckResponseSuccess(c, msg)
}

// sendWithoutSaveHandler handles request from client to publish a message without persisting it to
// database, handles verifying parsed data, error from topic manager, and sending response to client.
func (s *WebSocketServer) sendWithoutSaveHandler(c *network.Client, msg network.WebSocketMessage) {
	if msg.ParsedData == nil {
		s.AckResponseBadRequest(c, msg, fmt.Errorf("data payload could not be parsed"))
		return
	}

	// get the current schema for this topic
	isMatch, err := s.topicManager.IsSchemaMatch(msg.Topic, msg.ParsedData)
	if err != nil || !isMatch { // if we get an error, just blame it on client for now.
		s.AckResponseBadRequest(c, msg, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Making error channel for database to async give errors about persistence
	// back to handler to communicate that with the client.
	errCh := make(chan error, 1)
	go func() {
		select {
		case err := <-errCh:
			if err != nil {
				s.AckResponseDatabaseError(c, msg, err)
			}
		case <-ctx.Done():
			log.WithFields(log.Fields{
				"topic":      msg.Topic,
				"client":     c.Id,
				"message_id": msg.Id,
				"Action":     msg.Action,
			}).Warnf("DB write timeout for topic: %s, client: %s", msg.Topic, c.Id)
			s.AckResponseDatabaseError(c, msg, fmt.Errorf("timeout when persisting"))
		}
	}()

	if err := s.topicManager.Publish(ctx, msg, c, msg.ParsedData, errCh); err != nil {
		s.AckResponseError(c, msg, err)
	} else {
		s.AckResponseSuccess(c, msg)
	}
}

/*
	/* FUTURE HANDLERS
	"publishMany": s.TopicManager.SetManyTopics(),
	"sendWithoutSave": s.TopicManager.SendWithoutSave(),
	"deleteManyTopics": s.TopicManager.DeleteManyTopics(),
	"listWithPattern": s.TopicManager.ListWithPattern(),
*/
