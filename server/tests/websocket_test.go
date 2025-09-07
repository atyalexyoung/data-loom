package tests

import (
	"encoding/json"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type WebSocketMessage struct {
	Id         string          `json:"id"`
	Action     string          `json:"action"`
	Topic      string          `json:"topic,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	RequireAck bool            `json:"requireAck,omitempty"`
	ParsedData map[string]any  `json:"-"`
}

// Test subscribe to non-existing topic
func TestWebSocketServer(t *testing.T) {
	srv, cancel, url, db := startTestServer(t)
	defer cancel()    // triggers graceful shutdown
	defer srv.Close() // optional extra safety
	defer db.Close()
	time.Sleep(500 * time.Millisecond)

	header := http.Header{}
	header.Set("Authorization", "data-loom-api-key") // whatever your authToken expects
	c, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	msg := WebSocketMessage{
		Id:     "testClient1",
		Action: "subscribe", // use an action your server knows
		Topic:  "testTopic",
		// Data:       i shouldn't need data for subscriptions
		RequireAck: true,
	}

	if err := c.WriteJSON(msg); err != nil {
		t.Fatal(err)
	}

	_, resp, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Received: %s", resp)
}

// test subscribe to existing topic

// test message with invalid action

// test integration with multiple clients
func TestWebSocketIntegration(t *testing.T) {
	srv, cancel, url, db := startTestServer(t)
	defer cancel()    // triggers graceful shutdown
	defer srv.Close() // optional extra safety
	defer db.Close()
	time.Sleep(500 * time.Millisecond)

	header := http.Header{}
	header.Set("Authorization", "data-loom-api-key") // whatever your authToken expects

	client1, _, err := websocket.DefaultDialer.Dial(url, header)
	require.NoError(t, err)
	defer client1.Close()

	client2, _, err := websocket.DefaultDialer.Dial(url, header)
	require.NoError(t, err)
	defer client2.Close()

	sendAndReceive := func(t *testing.T, conn *websocket.Conn, msg WebSocketMessage) WebSocketMessage {
		require.NoError(t, conn.WriteJSON(msg))
		var resp WebSocketMessage
		require.NoError(t, conn.ReadJSON(&resp))
		return resp
	}

	t.Run("registerTopic", func(t *testing.T) {
		msg := WebSocketMessage{
			Id:         "1",
			Action:     "registerTopic",
			Topic:      "test-topic",
			Data:       json.RawMessage(`{"testKey": "testData"}`),
			RequireAck: true,
		}
		resp := sendAndReceive(t, client1, msg)
		assert.Equal(t, "1", resp.Id)
		logData(resp)
	})

	t.Run("subscribe client2", func(t *testing.T) {
		msg := WebSocketMessage{
			Id:         "2",
			Action:     "subscribe",
			Topic:      "test-topic",
			RequireAck: true,
		}
		resp := sendAndReceive(t, client2, msg)
		assert.Equal(t, "2", resp.Id)
		log.Printf("recieved: %+v", resp)
	})

	t.Run("publish message", func(t *testing.T) {
		msg := WebSocketMessage{
			Id:         "3",
			Action:     "publish",
			Topic:      "test-topic",
			Data:       json.RawMessage(`{"testKey": "testData"}`),
			RequireAck: true,
		}
		_ = sendAndReceive(t, client1, msg)

		// client2 should receive published message
		var recv WebSocketMessage
		require.NoError(t, client2.ReadJSON(&recv))
		logData(recv)
		log.Printf("\tmsg.Id : %s\n", recv.Id)
		log.Printf("\tmsg.Action : %s\n", recv.Action)
		log.Printf("\tmsg.Topic : %s\n", recv.Topic)
		log.Printf("\tmsg.Data : %s\n", recv.Data)
		assert.Equal(t, "test-topic", recv.Topic)
		assert.JSONEq(t, `{"testKey": "testData"}`, string(recv.Data))

		time.Sleep(500 * time.Millisecond)

	})

	t.Run("get last message", func(t *testing.T) {
		time.Sleep(5 * time.Second)
		msg := WebSocketMessage{
			Id:         "4",
			Action:     "get",
			Topic:      "test-topic",
			RequireAck: true,
		}
		resp := sendAndReceive(t, client2, msg)
		assert.Equal(t, "4", resp.Id)
		assert.JSONEq(t, `{"testKey": "testData"}`, string(resp.Data))
		logData(resp)
	})
}

func logData(msg WebSocketMessage) {
	if len(msg.Data) > 0 {
		var prettyData map[string]any
		if err := json.Unmarshal(msg.Data, &prettyData); err != nil {
			log.Printf("-------------------------------------------------------Failed to unmarshal Data: %v", err)
		} else {
			b, _ := json.MarshalIndent(prettyData, "", "  ")
			log.Printf("----------------------------------------------------Data: %s", b)
		}
	} else {
		log.Printf("---------------------------------------------------msg.Data was empty!!!!")
	}
}
