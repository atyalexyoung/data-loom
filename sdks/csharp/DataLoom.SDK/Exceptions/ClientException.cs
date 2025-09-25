namespace DataLoom.SDK.Exceptions
{
    public class ClientException : Exception
    {
        public ClientException(string message) : base(message) { }
        public ClientException(string message, Exception inner) : base(message, inner) { }
    }

    // Thrown when subscription fails
    public class SubscriptionFailedException : ClientException
    {
        public string Topic { get; }
        public int? ServerCode { get; }

        public SubscriptionFailedException(string topic, string message, int? serverCode = null)
            : base(message)
        {
            Topic = topic;
            ServerCode = serverCode;
        }
    }

    // Thrown when a request times out
    public class ResponseTimeoutException : ClientException
    {
        public string RequestId { get; }

        public ResponseTimeoutException(string requestId, string message)
            : base(message)
        {
            RequestId = requestId;
        }
    }    
}
