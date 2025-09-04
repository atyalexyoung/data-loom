package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"runtime"

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
		s.SendToClient(c, network.Response{
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
	s.SendToClient(c, network.Response{
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
	s.SendToClient(c, network.Response{
		Id:      msg.Id,
		Type:    msg.Action,
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
	log.Infof("Registering handler for action=%s, function=%s", action, name)

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

// publishHandler handles getting the request to publish from a client, error handling
// from trying to publish, and response to the sending client.
func (s *WebSocketServer) publishHandler(c *network.Client, msg network.WebSocketMessage) {
	if err := s.topicManager.Publish(msg.Topic, c, msg.Data); err != nil {
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

// unsubscribAllHandler handles the request from client to unsubscribe from all topics,
// and sending respone to the requesting client.
func (s *WebSocketServer) unsubscribeAllHandler(c *network.Client, msg network.WebSocketMessage) {
	s.topicManager.UnsubscribeAll(c)
	s.AckResponseSuccess(c, msg)
}

// getHandler handles a request to get a topic value, errors from topic manager, and
// sending response to requesting client.
func (s *WebSocketServer) getHandler(c *network.Client, msg network.WebSocketMessage) {
	if data, err := s.topicManager.Get(msg.Topic); err != nil {
		s.AckResponseError(c, msg, err)
	} else {
		s.AckResponseSuccessWithData(c, msg, data)
	}
}

// registerTopicHandler handles a request to register a topic, error from topic manager from
// operation, and sending response to the requesting client.
func (s *WebSocketServer) registerTopicHandler(c *network.Client, msg network.WebSocketMessage) {
	// get schema from message
	schema, err := parseJSON[map[string]any](msg.Data)
	if err != nil {
		s.AckResponseError(c, msg, err)
		return
	}

	topic, err := s.topicManager.RegisterTopic(msg.Topic, schema)
	if err != nil {
		s.AckResponseError(c, msg, err)
	} else if msg.RequireAck { // explicit check for requireAck since response with data doesn't
		s.AckResponseSuccessWithData(c, msg, topic)
	}
}

// unregisterTopicHandler handles request from client to unregister a topic, error from topic
// manager doing work, and responding to the requesting client.
func (s *WebSocketServer) unregisterTopicHandler(c *network.Client, msg network.WebSocketMessage) {
	if err := s.topicManager.UnregisterTopic(msg.Topic); err != nil {
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
// converting the raw json to map[string]any for storage, and errors from
// topic manager perfroming actions, and sending response to client.
// msg.Data known to not be null here from decorators.
func (s *WebSocketServer) updateSchemaHandler(c *network.Client, msg network.WebSocketMessage) {

	// validate schema and get map
	var newSchema map[string]any
	if err := json.Unmarshal(msg.Data, &newSchema); err != nil {
		s.AckResponseError(c, msg, err)
		return
	}

	// update schema and respond
	err := s.topicManager.UpdateSchema(msg.Topic, newSchema)
	if err != nil {
		s.AckResponseError(c, msg, err)
		return
	}

	s.AckResponseSuccess(c, msg)
}

func (s *WebSocketServer) sendWithoutSaveHandler(c *network.Client, msg network.WebSocketMessage) {
	if err := s.topicManager.Publish(msg.Topic, c, msg.Data); err != nil {
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
