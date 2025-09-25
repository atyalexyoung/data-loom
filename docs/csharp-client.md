# Data Loom C# Client

The C# SDK for Data Loom provides an abstraction over the WebSocket API for interacting with the server. It handles connecting, subscribing, publishing, and managing topics, while also ensuring schema validation and optional persistence.

> Note: This SDK is currently untested and planned to be distributed as a NuGet package in the future so the setup and usage will change.

## Prerequisites

- .NET 8 SDK installed
- Access to a running Data Loom server (see Getting Started)

## Installation

Currently, the SDK is not published as a package. To use it:

1. Clone the repository.
2. Build the SDK project located at `/sdks/csharp/DataLoom.SDK/`.
3. Add a reference to the compiled library in your C# project.

## Basic Concepts

### MessagingClientBuilder

The SDK uses a builder pattern to configure and create a messaging client. The builder allows fluent configuration of:

- Server URL
- API Key
- Client ID
- Whether to receive server acknowledgements
- Number of reconnect retries

But the only required options are the Server URL and API Key. It is planned update this to have the API Key as not required. 

After building the client, you receive an `IMessagingClient` instance to interact with the server.

### Topics

Topics are the core concept in Data Loom:

- Publishers send messages to a topic.
- Subscribers receive messages in real-time.
- Schema validation ensures published messages conform to the topic’s defined structure.

## Creating a Client

Use the `MessagingClientBuilder` to configure and create your client:

```csharp
using DataLoom.SDK.Clients;
using DataLoom.SDK.Builders;
using DataLoom.SDK.Interfaces;

var client = new MessagingClientBuilder()
    .WithServerUrl("ws://localhost:8080/ws")
    .WithApiKey("your-api-key")
    .WithClientId("my-client-id")
    .WithAckFromServer(true)
    .WithReconnectRetried(3)
    .Build();
```

The returned `client` implements `IMessagingClient` and provides all the methods for interacting with the server.

## Connecting to the Server

Once built, connect asynchronously:

```csharp
await client.ConnectAsync();
```

The SDK will handle incoming messages in the background.

## Publishing Messages

To publish data to a topic:

```csharp
await client.PublishAsync("chat-room", new { Message = "Hello, world!" });
```

To send a message without persisting it:

```csharp
await client.SendWithoutSaveAsync("chat-room", new { Message = "Ephemeral message" });
```

## Subscribing to Topics

Subscribe to a topic using a callback to handle incoming messages:

```csharp
var token = await client.SubscribeAsync<dynamic>("chat-room", async (msg) =>
{
    Console.WriteLine($"Received: {msg.Data.Message}");
});
```

Unsubscribe from a specific topic:

```csharp
await client.UnsubscribeAsync("chat-room", token);
```

Unsubscribe from all topics:

```csharp
await client.UnsubscribeAllAsync();
```

## Topic Management

### Register a Topic

```csharp
await client.RegisterTopicAsync<MyTopicSchema>("chat-room");
```

### Update a Topic Schema

```csharp
await client.UpdateSchemaAsync<NewSchema>("chat-room");
```

### Unregister a Topic

```csharp
await client.UnregisterTopicAsync("chat-room");
```

### List Topics

```csharp
var topics = await client.ListTopicsAsync();
foreach (var t in topics)
{
    Console.WriteLine(t);
}
```

### Get Current Value

```csharp
var value = await client.GetAsync<MyTopicSchema>("chat-room");
```

## Handling Errors

- `ServerException`: Indicates a server-side error (500 Internal Server Error).
- `ResponseTimeoutException`: Raised if a response from the server times out.
- `SubscriptionFailedException`: Raised if a subscription cannot be managed locally.
- `ArgumentException`: Raised for invalid input (e.g., invalid topic name).

All server responses contain a `code` and `message` field, following HTTP-style status codes:

- `200 OK`: Successful operation
- `400 Bad Request`: Malformed client request
- `500 Internal Server Error`: Server-side error or persistence failure

## Example Usage

```csharp
using DataLoom.SDK.Clients;
using DataLoom.SDK.Builders;
using DataLoom.SDK.Interfaces;

var client = new MessagingClientBuilder()
    .WithServerUrl("ws://localhost:8080/ws")
    .WithApiKey("my-key")
    .WithClientId("my-client-id")
    .WithAckFromServer(true)
    .Build();

await client.ConnectAsync();
await client.RegisterTopicAsync<MyTopicSchema>("chat-room");

var token = await client.SubscribeAsync<MyTopicSchema>("chat-room", async (msg) =>
{
    Console.WriteLine($"New message: {msg.Data.Content}");
});

await client.PublishAsync("chat-room", new MyTopicSchema { Content = "Hello everyone!" });

var value = await client.GetAsync<MyTopicSchema>("chat-room");

await client.UnsubscribeAsync("chat-room", token);
await client.UnregisterTopicAsync("chat-room");
```

## Notes

- All methods are asynchronous.
- The builder ensures correct configuration before connecting.
- The SDK handles serialization, deserialization, and schema validation.
- Ensure `ClientId` is unique for each client instance to avoid connection rejection.
