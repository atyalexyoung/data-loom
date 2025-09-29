using System.Text.Json;
using DataLoom.SDK.Models;

namespace DataLoom.SDK.Subscriptions
{
    internal class Subscription<T> : ISubscription
    {
        public string TopicName { get; set; }
        public Func<WebSocketMessage<T>, Task> Callback { get; }
        public SubscriptionToken SubscriptionToken { get; set; }

        public Subscription(string topicName, Func<WebSocketMessage<T>, Task> callback)
        {
            TopicName = topicName;
            Callback = callback;
            SubscriptionToken = new SubscriptionToken();
        }

        public async Task InvokeAsync(WebSocketMessage<JsonElement> message)
        {
            T? typedData = default;
            if (message.Data.ValueKind != JsonValueKind.Undefined && message.Data.ValueKind != JsonValueKind.Null)
            {
                try
                {
                    typedData = message.Data.Deserialize<T>();
                }
                catch (Exception ex)
                {
                    // Optional: log deserialization error, decide whether to call callback or not
                    throw new InvalidOperationException($"Failed to deserialize subscription payload to {typeof(T)}", ex);
                }
            }

            var typedMsg = new WebSocketMessage<T>
            {
                MessageId = message.MessageId,
                SenderId = message.SenderId,
                Action = message.Action,
                Topic = message.Topic,
                Data = typedData,
                RequireAck = message.RequireAck
            };

            await Callback(typedMsg);
        }
    }
}
