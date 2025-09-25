using System.Collections.Concurrent;
using System.Net;
using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using DataLoom.SDK.Builders;
using DataLoom.SDK.Exceptions;
using DataLoom.SDK.interfaces;
using DataLoom.SDK.Models;
using DataLoom.SDK.Subscriptions;

namespace DataLoom.SDK.Clients
{
    public class MessagingClient : IMessagingClient
    {
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

        private MessagingClientOptions _options;
        private readonly ClientWebSocket _webSocket = new();
        private ConcurrentDictionary<string, TaskCompletionSource<WebSocketResponse>> _pendingResponses = new();
        private SubscriptionManager _subscriptionManager = new();

        internal MessagingClient(MessagingClientOptions options)
        {
            _options = options ?? throw new ArgumentNullException(nameof(options));
            if (string.IsNullOrWhiteSpace(_options.ServerUrl))
            {
                throw new InvalidOperationException("Server URL cannot be null or whitespace.");
            }
            if (string.IsNullOrWhiteSpace(_options.ApiKey))
            {
                throw new InvalidOperationException("API key cannot be null or whitespace.");
            }
        }

        public async Task ConnectAsync()
        {
            var uri = new Uri(_options.ServerUrl!);
            _webSocket.Options.SetRequestHeader("Authorization", _options.ApiKey);
            _webSocket.Options.SetRequestHeader("ClientId", _options.ClientId);
            await _webSocket.ConnectAsync(uri, CancellationToken.None);

            _ = Task.Run(ReceiveLoopAsync);
        }

        private async Task ReceiveLoopAsync()
        {
            var buffer = new byte[8192];

            while (_webSocket.State == WebSocketState.Open)
            {
                var result = await _webSocket.ReceiveAsync(buffer, CancellationToken.None);
                if (result.MessageType == WebSocketMessageType.Close) { break; }

                var json = Encoding.UTF8.GetString(buffer, 0, result.Count);

                if (TryDeserialize<WebSocketResponse>(json, out var response))
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
        }

        public async Task<SubscriptionToken> SubscribeAsync<T>(string topicName, Func<WebSocketMessage<T>, Task> onMessageReceivedCallback)
        {
            Subscription<T> subscription = new Subscription<T>(topicName, onMessageReceivedCallback);
            var token = _subscriptionManager.AddSubscription(subscription);

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                Action = SUBSCRIBE,
                Topic = topicName,
                Id = Guid.NewGuid().ToString(),
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

        public async Task PublishAsync<T>(string topicName, T value)
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<T>
            {
                Id = Guid.NewGuid().ToString(),
                Action = PUBLISH,
                Topic = topicName,
                Data = value,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "publishing to topic: " + topicName);
        }


        public async Task UnsubscribeAsync(string topicName, SubscriptionToken subscriptionToken)
        {
            if (!_subscriptionManager.TryRemoveSubscription(topicName, subscriptionToken))
            {
                throw new SubscriptionFailedException(topicName, "Could not remove subscription from dictionary on client.");
            }

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                Id = Guid.NewGuid().ToString(),
                Action = UNSUBSCRIBE,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "Unsubscribing from topic: " + topicName);
        }

        public async Task UnsubscribeAllAsync()
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                Id = Guid.NewGuid().ToString(),
                Action = UNSUBSCRIBE_ALL,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "unsubscribing from all topics.");
        }

        public async Task<T?> GetAsync<T>(string topicName)
        {
            var response = await SendAndWaitForAckAsync(new WebSocketMessage<T>
            {
                Action = GET,
                Topic = topicName,
                Id = Guid.NewGuid().ToString(),
                RequireAck = true
            }, true) ?? throw new ServerException(-1, "Recieved null response from server from get request for topic: " + topicName);

            ValidateResponse(response, "get request for topic: " + topicName);

            // Convert to typed response
            var typedResponse = new WebSocketResponse<T>(response);
            return typedResponse.Data;
        }


        public async Task RegisterTopicAsync<T>(string topicName)
        {
            var schema = typeof(T)
                .GetProperties()
                .ToDictionary(p => p.Name, p => default(object));

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                Id = Guid.NewGuid().ToString(),
                Action = REGISTER_TOPIC,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled,
                Data = schema
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "registering topic: " + topicName);
        }

        public async Task UnregisterTopicAsync(string topicName)
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                Id = Guid.NewGuid().ToString(),
                Action = UNREGISTER_TOPIC,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "unregistering topic: " + topicName);
        }

        public async Task<IEnumerable<string>> ListTopicsAsync()
        {
            var response = await SendAndWaitForAckAsync(new WebSocketMessage<object?>
            {
                Action = LIST_TOPICS,
                Id = Guid.NewGuid().ToString(),
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


        public async Task UpdateSchemaAsync<T>(string topicName)
        {
            var schema = typeof(T)
                .GetProperties()
                .ToDictionary(p => p.Name, p => default(object));

            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<object>
            {
                Id = Guid.NewGuid().ToString(),
                Action = UPDATE_SCHEMA,
                Topic = topicName,
                RequireAck = _options.IsServerAckEnabled,
                Data = schema
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "updating schema for: " + topicName);
        }

        public async Task SendWithoutSaveAsync<T>(string topicName, T value)
        {
            var ack = await SendAndWaitForAckAsync(new WebSocketMessage<T>
            {
                Id = Guid.NewGuid().ToString(),
                Action = SEND_WITHOUT_SAVE,
                Topic = topicName,
                Data = value,
                RequireAck = _options.IsServerAckEnabled,
            }, _options.IsServerAckEnabled);

            ValidateResponse(ack, "sendWithoutSave to topic: " + topicName);
        }

        private TaskCompletionSource<WebSocketResponse> NewTcs(string requestId)
        {
            var tcs = new TaskCompletionSource<WebSocketResponse>(TaskCreationOptions.RunContinuationsAsynchronously);
            if (!_pendingResponses.TryAdd(requestId, tcs))
                throw new InvalidOperationException($"Duplicate request id: {requestId}");
            return tcs;
        }

        private async Task<WebSocketResponse?> SendAndWaitForAckAsync<T>(WebSocketMessage<T> request, bool requiresResponse, int timeoutMs = 5000)
        {
            if (!requiresResponse)
            {
                await SendWebSocketMessage(request);
                return null;
            }

            var tcs = NewTcs(request.Id);
            await SendWebSocketMessage(request);

            var delayTask = Task.Delay(timeoutMs);
            var completedTask = await Task.WhenAny(tcs.Task, delayTask).ConfigureAwait(false);

            if (completedTask == delayTask)
            {
                // timeout: remove the pending TCS so it doesn't leak
                _pendingResponses.TryRemove(request.Id, out _);
                throw new ResponseTimeoutException(request.Id, $"Response timed out for request {request.Id}");
            }
            return await tcs.Task;
        }

        private Task SendWebSocketMessage<T>(WebSocketMessage<T> message)
        {
            string json = JsonSerializer.Serialize(message);
            var bytes = Encoding.UTF8.GetBytes(json);
            return _webSocket.SendAsync(new ArraySegment<byte>(bytes), WebSocketMessageType.Text, true, CancellationToken.None);
        }

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

        private static void ValidateResponse(WebSocketResponse? response, string context)
        {
            if (response == null)
                throw new ServerException(-1, $"Received null response from server: {context}");

            if (response.Code != (int)HttpStatusCode.OK)
                throw new ServerException(response.Code, response.Message ?? $"Server error: {context}");
        }
    }
}