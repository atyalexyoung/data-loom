using DataLoom.SDK.Builders;
using DataLoom.SDK.Interfaces;

namespace IntegrationTests.Helpers
{
    public static class TestClientFactory
    {
        public static IMessagingClient BuildClient(string clientId)
        {
            return new MessagingClientBuilder()
                //.WithApiKey("") // no API key needed in default configuration of server.
                .WithServerUrl("ws://localhost:8080/ws")
                .WithClientId(clientId)
                .WithAckFromServer(true)
                .Build();
        }
    }
}
