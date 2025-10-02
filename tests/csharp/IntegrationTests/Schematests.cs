using IntegrationTests.Helpers;
using IntegrationTests.Models;

namespace IntegrationTests
{
    public class SchemaTests
    {
        [Fact]
        public async Task Updating_Topic_Schema_Succeeds()
        {
            var client = TestClientFactory.BuildClient("schema-client");
            await client.ConnectAsync();
            await client.RegisterTopicAsync<ChatMessage>("chat-room");

            // Pretend new schema type
            await client.UpdateSchemaAsync<NewChatMessage>("chat-room");

            var topics = await client.ListTopicsAsync();
            Assert.Contains("chat-room", topics);
        }

    }
}
