package server

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
)

// injectSenderIdDecorator will insert the client ID that the server has for a client into the message.
func (s *WebSocketServer) injectSenderIdDecorator(next HandlerFunc) HandlerFunc {
	return func(c *network.Client, msg network.WebSocketMessage) {
		// Overwrite any client-sent senderId with the server-known client Id
		msg.SenderId = c.Id
		next(c, msg)
	}
}

func (s *WebSocketServer) requireTopicDecorator(next HandlerFunc) HandlerFunc {
	log.Trace("Returning require message topic decorator.")
	return func(c *network.Client, msg network.WebSocketMessage) {
		if len(strings.TrimSpace(msg.Topic)) == 0 {
			s.AckResponseBadRequest(c, msg, fmt.Errorf("no topic provided"))
			return
		}

		log.WithFields(msg.GetLogFields()).
			WithField("client", c.Id).
			Trace("Require Topic Passed")

		next(c, msg)
	}
}

// requireDataDecorator will verify that the message has a "Data" field, and that
// it has some data in it.
func (s *WebSocketServer) requireDataDecorator(next HandlerFunc) HandlerFunc {
	log.Trace("Returning require message data decorator.")
	return func(c *network.Client, msg network.WebSocketMessage) {
		if msg.Data == nil { // check if null
			s.AckResponseBadRequest(c, msg, fmt.Errorf("message data was null"))
			return
		}
		if len(msg.Data) == 0 { // check if empty
			s.AckResponseBadRequest(c, msg, fmt.Errorf("message data was empty"))
			return
		}
		payload, err := parseJSON[map[string]any](msg.Data)
		if err != nil {
			s.AckResponseBadRequest(c, msg, err)
			return
		}
		msg.ParsedData = payload

		log.WithFields(msg.GetLogFields()).
			WithField("client", c.Id).
			Trace("Require Data Passed")

		next(c, msg) // if we good, then move along.
	}
}

func (s *WebSocketServer) metricsDecorator(next HandlerFunc) HandlerFunc {
	log.Trace("Returning metrics decorator")
	return func(c *network.Client, msg network.WebSocketMessage) {
		start := time.Now()
		log.WithFields(msg.GetLogFields()).
			WithField("client", c.Id).
			WithField("time", start).
			Debug("Started Metrics Decorator")

		next(c, msg)

		duration := time.Since(start)

		log.WithFields(msg.GetLogFields()).
			WithField("client", c.Id).
			WithField("duration", duration).
			Debug("Metrics Decorator Complete")
	}
}
