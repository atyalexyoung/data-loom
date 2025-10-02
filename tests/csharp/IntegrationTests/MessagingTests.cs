using IntegrationTests.Helpers;
using IntegrationTests.Models;

namespace IntegrationTests
{
    public class MessagingTests
    {
        [Fact]
        public async Task Subscribe_Receives_Published_Message()
        {
            var client = TestClientFactory.BuildClient("subscriber-client");
            await client.ConnectAsync();
            await client.RegisterTopicAsync<ChatMessage>("chat-room");

            var tcs = new TaskCompletionSource<ChatMessage>();

            await client.SubscribeAsync<ChatMessage>("chat-room", msg => {
                tcs.TrySetResult(msg.Data);
                return Task.CompletedTask;
            });

            var outgoing = new ChatMessage { Sender = "tester", Message = "ping" };
            await client.PublishAsync("chat-room", outgoing);

            var received = await Task.WhenAny(tcs.Task, Task.Delay(2000));
            Assert.True(received == tcs.Task, "Message was not received in time");
            Assert.Equal("ping", tcs.Task.Result.Message);
        }

    }
}
