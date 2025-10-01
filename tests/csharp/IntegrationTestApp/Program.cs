using System;
using System.Threading.Tasks;
using DataLoom.SDK.Builders;
using DataLoom.SDK.Interfaces;

namespace IntegrationTestApp
{
    // Define the type to use for the topic schema and messages
    public class ChatMessage
    {
        public string Sender { get; set; } = "";
        public string Message { get; set; } = "";
        public DateTime Timestamp { get; set; } = DateTime.UtcNow;
    }

    internal class Program
    {
        private static async Task Main(string[] args)
        {
            Console.WriteLine("Starting DataLoom integration test with ChatMessage...");

            IMessagingClient client = new MessagingClientBuilder()
                .WithApiKey("data-loom-api-key")
                .WithServerUrl("ws://localhost:8080/ws") // adjust to your server
                .WithClientId("chat-integration-test")
                .WithAckFromServer(true)
                .Build();

            try
            {
                await client.ConnectAsync();
                Console.WriteLine("Connected!");
                await Task.Delay(1000);

                // Register a topic using the ChatMessage type as schema
                var topicSchema = new ChatMessage(); // instance used to infer schema
                await client.RegisterTopicAsync<ChatMessage>("chat-room");
                Console.WriteLine("Registered topic 'chat-room'.");

                await Task.Delay(1000);

                // Subscribe to the topic
                var token = await client.SubscribeAsync<ChatMessage>("chat-room", async (message) =>
                {
                    Console.WriteLine($"[{message.Topic }]: Action: {message.Action}. Message: {message.Data.Message} from {message.Data.Sender} at {message.Data.Timestamp}");
                    
                    await Task.CompletedTask;
                });
                Console.WriteLine("Subscribed to 'chat-room'.");

                await Task.Delay(1000);

                // Publish a message
                var outgoingMessage = new ChatMessage
                {
                    Sender = "IntegrationTestClient",
                    Message = "Hello everyone!",
                    Timestamp = DateTime.UtcNow
                };

                await client.PublishAsync("chat-room", outgoingMessage);
                Console.WriteLine("Published message.");

                // Allow time for messages to be received
                await Task.Delay(1000);

                // Cleanup
                await client.UnsubscribeAsync("chat-room", token);
                Console.WriteLine("Unsubscribed.");

                await Task.Delay(1000);

                await client.DisconnectAsync();
                Console.WriteLine("Disconnected.");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Integration test failed: {ex.Message}");
            }

            Console.WriteLine("Test complete. Press any key to exit.");
            //Console.ReadKey();
        }
    }
}
