using System.Text.Json;
using DataLoom.SDK.Models;

namespace DataLoom.SDK.Subscriptions
{
    public interface ISubscription
    {
        string TopicName { get; }
        Task InvokeAsync(WebSocketMessage<JsonElement> message);
        SubscriptionToken SubscriptionToken { get; }
    }
}