namespace DataLoom.SDK.Subscriptions
{
    public class SubscriptionToken
    {
        string Id { get; init; } = Guid.NewGuid().ToString();
    }

}