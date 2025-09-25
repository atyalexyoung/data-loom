using System.Collections.Concurrent;
using System.Text.Json;
using DataLoom.SDK.Models;

namespace DataLoom.SDK.Subscriptions
{
    internal class SubscriptionManager
    {
        private readonly ConcurrentDictionary<string, ConcurrentDictionary<SubscriptionToken, ISubscription>> _subscriptions = new();

        public SubscriptionToken AddSubscription(ISubscription subscription)
        {
            var dict = _subscriptions.GetOrAdd(subscription.TopicName, _ => new ConcurrentDictionary<SubscriptionToken, ISubscription>());
            dict[subscription.SubscriptionToken] = subscription;
            return subscription.SubscriptionToken;
        }

        public bool TryRemoveSubscription(string topic, SubscriptionToken token)
        {
            if (_subscriptions.TryGetValue(topic, out var dict))
            {
                return dict.TryRemove(token, out _);
            }
            return false;
        }

        public async Task DispatchAsync(WebSocketMessage<JsonElement> message)
        {
            if (message.Topic != null && _subscriptions.TryGetValue(message.Topic, out var dict))
            {
                foreach (var sub in dict.Values)
                {
                    _ = sub.InvokeAsync(message);
                }
            }
        }
    }
}