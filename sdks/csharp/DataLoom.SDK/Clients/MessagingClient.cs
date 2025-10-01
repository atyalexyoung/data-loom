using System.Collections.Concurrent;
using System.Net;
using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using DataLoom.SDK.Builders;
using DataLoom.SDK.Exceptions;
using DataLoom.SDK.Interfaces;
using DataLoom.SDK.Models;
using DataLoom.SDK.Subscriptions;

namespace DataLoom.SDK.Clients
{
    public class MessagingClient : IMessagingClient
    {
        // Constants declared for message type field being sent to server.

        private const string SUBSCRIBE = "subscribe";
        private const string PUBLISH = "publish";
        private const string UNSUBSCRIBE = "unsubscribe";
        private const string UNSUBSCRIBE_ALL = "unsubscribeAll";
        private const string GET = "get";
        private const string REGISTER_TOPIC = "registerTopic";
        private const string UNREGISTER_TOPIC = "unregisterTopic";
        private const string LIST_TOPICS = "listTopics";
        private const string UPDATE_SCHEMA = "updateSchema";
        private const string SEND_WITHOUT_SAVE = "sendWithoutSave";

        /// <summary>
        /// Options for all the configurations for the client and connection to server.
        /// </summary>
        private MessagingClientOptions _options;

        /// <summary>
        /// Dictionary of the message ID mapped to task completion source to handle responses from server for particular messages.
        /// </summary>
        private ConcurrentDictionary<string, TaskCompletionSource<WebSocketResponse>> _pendingResponses = new();

        /// <summary>
        /// Subscription Managaer that will handle subscribing, unsubscribing and sending messages to all subscribers.
        /// </summary>
        private SubscriptionManager _subscriptionManager = new();

        /// <summary>
        /// Websocket connection from the client to the server.
        /// </summary>
        private readonly ClientWebSocket _webSocket = new();

        /// <summary>
        /// Cancellation source for the receive loop.
        /// </summary> 
        private CancellationTokenSource _receiveLoopCts = new();

        /// <summary>
        /// Main constructor for MessagingClient. Takes MessagingClientOptions instance for configuration.
        /// </summary>
        /// <param name="options">The <see cref="MessagingClientOptions"/> for configuration of client.</param>
        /// <exception cref="ArgumentNullException">Thrown if options is null.</exception>
        /// <exception cref="InvalidOperationException">Thrown if the Server URL is null or whitespace, or if API key is null.</exception>
        internal MessagingClient(MessagingClientOptions options)
        {
            _options = options ?? throw new ArgumentNullException(nameof(options));
            if (string.IsNullOrWhiteSpace(_options.ServerUrl))
            {
                throw new InvalidOperationException("Server URL cannot be null or whitespace.");
            }
            // if the API key is null we will just not supply one.
        }


        /// <summary>
        /// Connects to the server with the configured URL, API Key and Client ID.k
        /// </summary>
        /// <returns>Task to do async work on.</returns>
        public async Task ConnectAsync()
        {
            var uri = new Uri(_options.ServerUrl!);
            if (_options.ApiKey != null)
            {
                _webSocket.Options.SetRequestHeader("Authorization", _options.ApiKey);
            }

            _webSocket.Options.SetRequestHeader("ClientId", _options.ClientId);
            await _webSocket.ConnectAsync(uri, CancellationToken.None);

            _receiveLoopCts = new CancellationTokenSource();

            _ = Task.Run(() => ReceiveLoopAsync(_receiveLoopCts));
        }
        
        /// <summary>
        /// Disconencts from the server websocket.
        /// </summary>
        /// <returns>Task to do async work on.</returns>
        /// <exception cref="WebSocketException">Thrown if there is an error when closing connection.</exception>
        public async Task DisconnectAsync()
        {
            try
                {
                    _receiveLoopCts?.Cancel();

                    if (_webSocket.State == WebSocketState.Open || _webSocket.State == WebSocketState.CloseReceived)
                    {
                        await _webSocket.CloseAsync(WebSocketCloseStatus.NormalClosure, "Client disconnecting", CancellationToken.None);
                    }
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Error during disconnect: {ex.Message}");
                }
                finally
                {
                    _webSocket.Dispose();
                }
        }

        /// <summary>
        /// Main receive loop for the websocket that handles and routes all incoming messages
        /// based on what they are (WebSocketResponse if a response from server, or 
        /// WebSocketMessage if published from another client).
        /// </summary>
        /// <returns>Task to do asynchronous work.</returns>
        private async Task ReceiveLoopAsync(CancellationTokenSource cts)
        {
            var buffer = new byte[8192];

            while (_webSocket.State == WebSocketState.Open && !cts.IsCancellationRequested)
            {
                var result = await _webSocket.ReceiveAsync(buffer, CancellationToken.None);
                if (result.MessageType == WebSocketMessageType.Close) { break; }

                var json = Encoding.UTF8.GetString(buffer, 0, result.Count);

                if (TryDeserialize<WebSocketResponse>(json, out var response) && response != null && response.Type == "response")
                {
                    if (response != null && !string.IsNullOrWhiteSpace(response.Id) &&
                        _pendingResponses.TryRemove(response.Id, out var tcs))
                    {
                        try
                        {
                            tcs.SetResult(response);
                        }
                        catch
                        {
                            // the timeout already happened so log that server too slow
                        }
                    }
                    continue;
                }

                if (TryDeserialize<WebSocketMessage<JsonElement>>(json, out var msg))
                {
                    if (msg == null || string.IsNullOrWhiteSpace(msg.Topic))
                    {
                        // log that null message received
                        continue;
                    }

                    await _subscriptionManager.DispatchAsync(msg!);
                    continue;
                }

                // log that message was recieved but couldn't be read.
            }

            // log that the websocket was not open or cancellation was requested.
        }

        /// <summary>
        /// Subscribes to a topic provided that routes publications to the callback that is provided.
        /// </summary>
        /// <typeparam name="T">The type of the value that will be on a particular topic.</typeparam>
        /// <param name="topicName">The name of the topic to be subscribed to.</param>
        /// <param name="onMessageReceivedCallback">The handler for an update to a topic.</param>
        /// <returns>Task to do async work with a subscription token. This token is to match a particular subscription to the handler.
        /// This approach allows anonymous methods to still be unsubscribed from.</returns>
        /// <exception cref="ServerException">Thrown if the server doesn't return success code for subscription operation.</exception>
        public async Task<SubscriptionToken> SubscribeAsync<T>(string topicName, Func<WebSocketMessage<T>, Task> onMessageReceivedCallback)
        {
            Subscription<T> subscription = new Subscription<T>(topicName, onMessageReceivedCallback);
            var token = _subscriptionManager.AddSubscription(subscription);

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                Action = SUBSCRIBE,
                Topic = topicName,
                MessageId = Guid.NewGuid().ToString(),
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            if (ack != null && ack.Code != (int)HttpStatusCode.OK)
            {
                if (_subscriptionManager.TryRemoveSubscription(topicName, token))
                {
                    // log success
                }

                // throw from server.
                throw new ServerException(ack.Code, ack.Message ?? "No message provided from server exception when subscribing.");
            }

            return subscription.SubscriptionToken;
        }

        /// <summary>
        /// Publishes a value to a specified topic.
        /// </summary>
        /// <typeparam name="T">The type of the topic that is being published to.</typeparam>
        /// <param name="topicName">The name of the topic that is being published to.</param>
        /// <param name="value">The value being published on the topic.</param>
        /// <returns>Task to do async work.</returns>
        public async Task PublishAsync<T>(string topicName, T value)
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<T>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = PUBLISH,
                Topic = topicName,
                Data = value,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "publishing to topic: " + topicName);
        }

        /// <summary>
        /// Unsubscribes from a topic.
        /// </summary>
        /// <param name="topicName">The topic to unsubscribe from.</param>
        /// <param name="subscriptionToken">The token of the handler to unsubscribe from that topic.</param>
        /// <returns>Task to do async work.</returns>
        /// <exception cref="SubscriptionFailedException">Thrown if there is an issue with removing the subscription.</exception>
        public async Task UnsubscribeAsync(string topicName, SubscriptionToken subscriptionToken)
        {
            if (!_subscriptionManager.TryRemoveSubscription(topicName, subscriptionToken))
            {
                throw new SubscriptionFailedException(topicName, "Could not remove subscription from dictionary on client.");
            }

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = UNSUBSCRIBE,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "Unsubscribing from topic: " + topicName);
        }

        /// <summary>
        /// Unsubscribes a client from all topics.
        /// </summary>
        /// <returns>Task to do async work.</returns>
        public async Task UnsubscribeAllAsync()
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = UNSUBSCRIBE_ALL,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "unsubscribing from all topics.");
        }

        /// <summary>
        /// Will get the current value for a particular topic.
        /// </summary>
        /// <typeparam name="T">The type of the value to be retrieved.</typeparam>
        /// <param name="topicName">The name of the topic to get the value for.</param>
        /// <returns>Task that contains a nullable instance of the value of the topic.</returns>
        /// <exception cref="ServerException">Thrown if server responds with null response or 
        /// a non-success response code. </exception>
        public async Task<T?> GetAsync<T>(string topicName)
        {
            var response = await SendAndWaitForAckAsync(new WebSocketMessage<T>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = GET,
                Topic = topicName,
                RequireAck = true
            }, true) ?? throw new ServerException(-1, "Recieved null response from server from get request for topic: " + topicName);

            ValidateResponse(response, "get request for topic: " + topicName);

            // Convert to typed response
            var typedResponse = new WebSocketResponse<T>(response);
            return typedResponse.Data;
        }

        /// <summary>
        /// Registers a topic with the name provided and schema of the type that is provided.
        /// </summary>
        /// <typeparam name="T">The type that will define the schema for the topic.</typeparam>
        /// <param name="topicName">The name of the topic to register.</param>
        /// <returns>Task to do async work.</returns>
        public async Task RegisterTopicAsync<T>(string topicName)
        {
            var schema = typeof(T)
                .GetProperties()
                .ToDictionary(p => p.Name, p => default(object));

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = REGISTER_TOPIC,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled,
                Data = schema
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "registering topic: " + topicName);
        }

        /// <summary>
        /// Unregisters a topic on the server for ALL clients.
        /// </summary>
        /// <param name="topicName">The name of the topic to unregister.</param>
        /// <returns>Task to do async work.</returns>
        public async Task UnregisterTopicAsync(string topicName)
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = UNREGISTER_TOPIC,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "unregistering topic: " + topicName);
        }

        /// <summary>
        /// Lists all the current topics that are registered on the server.
        /// </summary>
        /// <returns>Task with payload of an IEnumerable of strings that are 
        /// the names of the currently registered topics.</returns>
        /// <exception cref="ServerException">Thrown if the response or response
        ///  payload from the serer is null, or if a non-success code is sent.</exception>
        public async Task<IEnumerable<string>> ListTopicsAsync()
        {
            var response = await SendAndWaitForAckAsync(new WebSocketMessage<object?>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = LIST_TOPICS,
                RequireAck = true
            }, true) ?? throw new ServerException(-1, "Recieved null response from server from list topics request");

            ValidateResponse(response, "listing topics");

            // Convert to typed response
            var typedResponse = new WebSocketResponse<IEnumerable<string>>(response);
            if (typedResponse.Data == null)
            {
                throw new ServerException(-1, response.Message ?? "No data provided from server exception when listing topics.");
            }
            return typedResponse.Data;
        }

        /// <summary>
        /// Updates the schema for a topic.
        /// </summary>
        /// <typeparam name="T">The type that will be the new schema for the topic.</typeparam>
        /// <param name="topicName">The name of the topic to update the schema of.</param>
        /// <returns>Task to do async work.</returns>
        public async Task UpdateSchemaAsync<T>(string topicName)
        {
            var schema = typeof(T)
                .GetProperties()
                .ToDictionary(p => p.Name, p => default(object));

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = UPDATE_SCHEMA,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled,
                Data = schema
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "updating schema for: " + topicName);
        }

        /// <summary>
        /// Sends a value as if it was published but the server will not persist 
        /// the value, so the current value for the topic will not be changed.
        /// </summary>
        /// <typeparam name="T">The type of the topic being sent on.</typeparam>
        /// <param name="topicName">The name of the topic to send the message to.</param>
        /// <param name="value">The value to send on the topic.</param>
        /// <returns>Task to do async work.</returns>
        public async Task SendWithoutSaveAsync<T>(string topicName, T value)
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<T>
            {
                MessageId = Guid.NewGuid().ToString(),
                Action = SEND_WITHOUT_SAVE,
                Topic = topicName,
                Data = value,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "sendWithoutSave to topic: " + topicName);
        }

        /// <summary>
        /// Helper method that creates a new <see cref="TaskCompletionSource"/> 
        /// for waiting for a response from the server for a particular request.
        /// </summary>
        /// <param name="requestId"></param>
        /// <returns></returns>
        /// <exception cref="InvalidOperationException"></exception>
        private TaskCompletionSource<WebSocketResponse> NewTcs(string requestId)
        {
            var tcs = new TaskCompletionSource<WebSocketResponse>(TaskCreationOptions.RunContinuationsAsynchronously);
            if (!_pendingResponses.TryAdd(requestId, tcs))
                throw new InvalidOperationException($"Duplicate request id: {requestId}");
            return tcs;
        }

        /// <summary>
        /// Helper method that will handle whether a response from the server should be awaited for, and handling the
        /// setup for waiting for timeout or the response from the server.
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="request"></param>
        /// <param name="requiresResponse"></param>
        /// <param name="timeoutMs"></param>
        /// <returns></returns>
        /// <exception cref="ResponseTimeoutException"></exception>
        private async Task<WebSocketResponse?> SendAndWaitForAckAsync<T>(WebSocketMessage<T> request, bool requiresResponse, int timeoutMs = 5000)
        {
            if (!requiresResponse)
            {
                await SendWebSocketMessage(request);
                return null;
            }

            var tcs = NewTcs(request.MessageId);
            await SendWebSocketMessage(request);

            var delayTask = Task.Delay(timeoutMs);
            var completedTask = await Task.WhenAny(tcs.Task, delayTask).ConfigureAwait(false);

            if (completedTask == delayTask)
            {
                // timeout: remove the pending TCS so it doesn't leak
                _pendingResponses.TryRemove(request.MessageId, out _);
                throw new ResponseTimeoutException(request.MessageId, $"Response timed out for request {request.MessageId}");
            }
            return await tcs.Task;
        }

        /// <summary>
        /// Helper method to serialize and send a web socket message.
        /// </summary>
        /// <typeparam name="T">The type of the WebSocketMessage to be sent.</typeparam>
        /// <param name="message">The message to be sent.</param>
        /// <returns>Task to do async work.</returns>
        private Task SendWebSocketMessage<T>(WebSocketMessage<T> message)
        {
            string json = JsonSerializer.Serialize(message);
            var bytes = Encoding.UTF8.GetBytes(json);
            return _webSocket.SendAsync(new ArraySegment<byte>(bytes), WebSocketMessageType.Text, true, CancellationToken.None);
        }

        /// <summary>
        /// Will attempt to deserialize a message.
        /// </summary>
        /// <typeparam name="T">The type to try and deserialize the message to.</typeparam>
        /// <param name="json">The string json to try and deserialize.</param>
        /// <param name="result">The result of the deserializtion.</param>
        /// <returns>Boolean if the deserialization was successful or not.</returns>
        private static bool TryDeserialize<T>(string json, out T? result)
        {
            try
            {
                result = JsonSerializer.Deserialize<T>(json);
                return true;
            }
            catch
            {
                result = default;
                return false;
            }
        }

        /// <summary>
        /// Helper method that will validate a response from the server by checking if it
        /// is null, or if the code that was returned was not 200 OK.
        /// </summary>
        /// <param name="response">The response from the server.</param>
        /// <param name="context">What type of message the validation is for. For better logging in the case that an exception should be thrown.</param>
        /// <exception cref="ServerException">Thrown if the validation fails (something is wrong with the message).</exception>
        private static void ValidateResponse(WebSocketResponse? response, string context)
        {
            Console.WriteLine("Response: " + response);

            if (response == null)
                throw new ServerException(-1, $"Received null response from server: {context}");

            if (response.Code != (int)HttpStatusCode.OK)
                throw new ServerException(response.Code, response.Message ?? $"Server error: {context}");
        }
    }
}