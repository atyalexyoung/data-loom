package logging

import (
	log "github.com/sirupsen/logrus"

	"os"
)

func init() {
	// --- Formatter ---
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		PrettyPrint:     true, // set true for dev, false for production
	})

	// --- Level ---
	//log.SetLevel(log.InfoLevel) // change to DebugLevel for dev
	log.SetLevel(log.TraceLevel) // change to DebugLevel for dev

	// --- Output ---
	log.SetOutput(os.Stdout)

	// --- Optional: Hooks ---
	// log.AddHook(NewSentryHook()) // add Sentry, Datadog, etc. if desired
}

// Logs an error in a handler
func HandlerError(clientId string, action string, topic string, messageId string, err error) {
	log.WithFields(log.Fields{
		"client_id": clientId,
		"action":    action,
		"topic":     topic,
		"msg_id":    messageId,
	}).Error(err)
}

// Logs a successful action
func HandlerSuccess(clientId string, action string, topic string, messageId string) {
	log.WithFields(log.Fields{
		"client_id": clientId,
		"action":    action,
		"topic":     topic,
		"msg_id":    messageId,
	}).Info("Action completed successfully")
}

// Logs that an acknowledgment was sent
func HandlerAck(clientId string, action string, topic string, messageId string) {
	log.WithFields(log.Fields{
		"client_id": clientId,
		"action":    action,
		"topic":     topic,
		"msg_id":    messageId,
		"ack":       true,
	}).Info("Acknowledgment sent")
}

// Logs that an acknowledgment was sent
func HandlerAckWithData(clientId string, action string, topic string, messageId string, data any) {
	log.WithFields(log.Fields{
		"client_id": clientId,
		"action":    action,
		"topic":     topic,
		"msg_id":    messageId,
		"ack":       true,
		"data":      data,
	}).Info("Acknowledgment sent")
}
