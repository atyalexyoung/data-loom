using IntegrationTests.Helpers;
using IntegrationTests.Models;

namespace IntegrationTests
{
    public class ChatIntegrationTests
    {
        [Fact]
        public async Task Full_Chat_Roundtrip_Works()
        {
            var client = TestClientFactory.BuildClient("chat-integration-test");

            await client.ConnectAsync();
            await client.RegisterTopicAsync<ChatMessage>("chat-room");

            var tcs = new TaskCompletionSource<ChatMessage>();
            var token = await client.SubscribeAsync<ChatMessage>("chat-room", msg =>
            {
                tcs.TrySetResult(msg.Data);
                return Task.CompletedTask;
            });

            var outgoing = new ChatMessage
            {
                Sender = "IntegrationTestClient",
                Message = "Hello everyone!",
                Timestamp = DateTime.UtcNow
            };

            await client.PublishAsync("chat-room", outgoing);

            var completed = await Task.WhenAny(tcs.Task, Task.Delay(2000));
            Assert.True(completed == tcs.Task, "Message was not received in time");

            await client.UnsubscribeAsync("chat-room", token);
            await client.DisconnectAsync();
        }
    }
}
