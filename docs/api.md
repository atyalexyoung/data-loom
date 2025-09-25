# API and Connection

This describes the messages that are sent and recieved, and how they work.

## Connection
The server is at /ws on port 8080 by default (port configuration is planned). When connecting via websocket to the server, 2 things need to be provided in the header:

1. Authorization: {your-api-key}
    - This is the authorization header with the API key that was configured for the server. If no API key is configured for the server, it currently defaults to: "data-loom-api-key", but it is planned to default to not requiring authorization if no API key is configured.
2. ClientId: {your-client-id}
    - This will be the ID for your client. Currently, messages do not contain the Client ID, but it is planned to include so when a message is received, you can tell where it came from. If the Client ID you provide is already in use, the server will reject the connection as the ID has to be unique.



## API and Messages

All communication happens over a single WebSocket connection.  
Messages are JSON objects that match the following schema:

**Client -> Server**

**"Web Socket Message"**
```jsonc
{
  "id": "unique-request-id",
  "action": "subscribe",
  "topic": "chat-room",
  "data": { ... },          // optional, depends on action
  "requireAck": true        // optional, request a server ack
}
```

**Client <- Server**

**"Web Socket Response"**
```jsonc
{
  "id": "unique-request-id",
  "type": "subscribe",
  "code": 200,
  "message": "OK or error message",
  "data": { ... }                   // optional payload
}
```

Messages that have ACKs will not respond with ACKs if the requireAck field is false. The server will only send back an ACK response if this field is true.

ACKs from the server are the same format as responses. Messages that have responses reguardless of the requireAck field contain information in the data field and are the "get" and "listTopics" actions.


### Actions Overview

| Action           | Description                                           | Required Fields                 | Response Data (if any)          |
|------------------|-------------------------------------------------------|---------------------------------|---------------------------------|
| `subscribe`      | Subscribe to updates on a topic.                      | `id`, `action`, `topic`         | Ack or error                    |
| `publish`        | Publish data to a topic.                              | `id`, `action`, `topic`, `data` | Ack or error.                   |
| `unsubscribe`    | Unsubscribe from a specific topic.                    | `id`, `action`, `topic`         | Ack or error.                   |
| `unsubscribeAll` | Unsubscribe from all topics.                          | `id`, `action`, `topic`         | Ack or error.                   |
| `get`            | Retrieve the current value of a topic.                | `id`, `action`, `topic`         | Current data for the topic.     |
| `registerTopic`  | Register a new topic with optional schema/data.       | `id`, `action`, `topic`, `data` | Ack or error.                   |
| `unregisterTopic`| Unregister an existing topic.                         | `id`, `action`, `topic`         | Ack or error.                   |
| `listTopics`     | List all available topics.                            | `id`, `action`                  | Array of topics.                |
| `updateSchema`   | Update the schema of an existing topic.               | `id`, `action`, `topic`, `data` | Ack or error.                   |
| `sendWithoutSave`| Send a message to a topic without persisting it.      | `id`, `action`, `topic`, `data` | Ack or error.                   |

### Actions In More Detail


#### registerTopic and Schemas

When registering topics via the "registerTopic" command, the "data" field is expected to be json format of the type that you want to register the topic as. The server takes the json object that is passed, and keeps that as the "schema".

The server uses the json object as a schema and validates that messages being sent match the schema that the topic was registered under by comparing the fields of the json object. This means that you can pass the json with the correct fields and anything as the values, as the values will not be used.

An example of this would be:

```jsonc
{
  "id": "message-specific-uuid",
  "action": "registerTopic",
  "topic": "new-topic-name",
  "data": {
    "MyDataType": {
        "myList": [
            "anything",
            "you",
            "want",
        ],
        "myDatasInt": 0,
        "myDatasString" : "",
        "NestedThing":{
            "nestedThingId": "blah",
            "nestedThingString": "hmm"
        }
    }
  },          
  "requireAck": false
}
```

Then the server would validate that the structure of the json that the other client is sending matches this. The idea of the SDKs is to provide an abstraction from this to use a languages native type system.

#### subscribe

When subscribing to a topic, you will get the entire Web Socket Message that the publisher sent and will contain the same fields that any client uses to send messages with the structure of:

```jsonc
{
  "id": "unique-request-id",
  "action": "subscribe",
  "topic": "chat-room",
  "data": { ... },          // optional, depends on action
  "requireAck": true        // optional, request a server ack
}
```

This means that in order to get the updated topic information, you will have to access the "data" field.



### Errors and Status Codes
When the server ACKs to a message, in the message there will be a field for "code" and "message".

The codes that are used are a subset of HTTP status codes to make it easier to diagnose what the issue is, and the message is text that describes the error if applicable.

For now, there are only a few used which are:

#### 200 (OK)

This is the code if the operation was successful and nothing was detected as going wrong. If the 200 OK is sent there will be no "message" field associated.

Example response for 200 OK:

```jsonc
{
  "id": "unique-request-id",
  "type": "subscribe",
  "code": 200,
}
```

#### 500 (Internal Server Error)

This code is used if something went wrong on the server during the handling of a request. The "message" field will give more details about what went wrong. This code is also used if something went wrong when persisting data, but the "type" field will be "persist".

Example response for server error:

```jsonc
{
  "id": "unique-request-id",
  "type": "subscribe",
  "code": 500,
  "message": "topic doesn't exist for example-topic-name",
}
```

Example response for database persistence error:

```jsonc
{
  "id": "unique-request-id",
  "type": "persist",
  "code": 500,
  "message": "timeout when persisting",
}
```

#### 400 (Bad Request)

This code is used if a request from a client is received as malformed or invalid in some way. The "message" field wil give more details about what was wrong with the request.

Example response for 400 Bad Request:

```jsonc
{
  "id": "unique-request-id",
  "type": "registerTopic",
  "code": 400,
  "message": "data payload could not be parsed",
}
```
