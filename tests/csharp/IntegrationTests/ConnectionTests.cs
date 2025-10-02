using Xunit;
using System.Threading.Tasks;
using IntegrationTests.Helpers;

namespace IntegrationTests
{
    public class ConnectionTests
    {
        [Fact]
        public async Task Connects_And_Disconnects_Cleanly()
        {
            var client = TestClientFactory.BuildClient("connection-test");
            await client.ConnectAsync();
            Assert.True(client.IsConnected);

            await client.DisconnectAsync();
            Assert.False(client.IsConnected);
        }
    }
}
