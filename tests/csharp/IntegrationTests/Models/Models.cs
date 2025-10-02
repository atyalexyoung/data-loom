namespace IntegrationTests.Models
{
    public class ChatMessage
    {
        public string Sender { get; set; } = "";
        public string Message { get; set; } = "";
        public DateTime Timestamp { get; set; } = DateTime.UtcNow;
    }

    public class NewChatMessage
    {
        public string Sender { get; set; } = "";
        public string Message { get; set; } = "";
        public DateTime Timestamp { get; set; } = DateTime.UtcNow;
        public bool IsNew { get; set; }
    }

    public class AlertMessage
    {
        public string Level { get; set; }
        public string Message { get; set; }
        public DateTime Timestamp { get; set; }
    }

    public class MetricMessage
    {
        public string Name { get; set; }
        public double Value { get; set; }
        public DateTime Timestamp { get; set; }
    }

    public class LogMessage
    {
        public string Source { get; set; }
        public string Message { get; set; }
        public DateTime Timestamp { get; set; }
    }
}