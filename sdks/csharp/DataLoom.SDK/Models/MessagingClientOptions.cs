namespace DataLoom.SDK.Models
{
    /// <summary>
    /// Contains options available for a MessagingClient.
    /// </summary>
    internal class MessagingClientOptions
    {
        public string? ApiKey;
        public string? ServerUrl = "";
        public bool IsServerAckEnabled;
        public int NumRetries = 3;
        public string ClientId = Guid.NewGuid().ToString();
    }
}