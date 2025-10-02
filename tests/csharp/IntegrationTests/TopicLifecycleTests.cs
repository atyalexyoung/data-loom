using IntegrationTests.Helpers;
using IntegrationTests.Models;

namespace IntegrationTests
{
    public class TopicLifecycleTests
    {
        [Fact]
        public async Task Can_Register_And_List_Topic()
        {
            var client = TestClientFactory.BuildClient("topic-client");
            await client.ConnectAsync();

            await client.RegisterTopicAsync<ChatMessage>("chat-room");
            var topics = await client.ListTopicsAsync();

            Assert.Contains("chat-room", topics);
        }

    }
}
