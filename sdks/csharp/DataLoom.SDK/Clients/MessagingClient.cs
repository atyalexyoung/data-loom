using DataLoom.SDK.Builders;
using DataLoom.SDK.interfaces;
using DataLoom.SDK.Models;

namespace DataLoom.SDK.Clients
{    
    public class MessagingClient : IMessagingClient
    {
        private MessagingClientOptions _options;

        public void Connect()
        {
            // network stuff?
        }

        internal MessagingClient(MessagingClientOptions options)
        {
            _options = options;
        }

        public Task<T> GetAsync<T>(string topicName)
        {
            throw new NotImplementedException();
        }

        public Task<IEnumerable<string>> ListTopicsAsync()
        {
            throw new NotImplementedException();
        }

        public Task PublishAsync<T>(string topicName, T value)
        {
            throw new NotImplementedException();
        }

        public Task RegisterTopicAsync<T>(string topicName)
        {
            throw new NotImplementedException();
        }

        public Task SendWithoutSaveAsync<T>(string topicName, T value)
        {
            throw new NotImplementedException();
        }

        public Task<SubscriptionToken> SubscribeAsync<T>(string topicName, Func<T, Task> onMessageReceivedCallback)
        {
            throw new NotImplementedException();
        }

        public Task UnregisterTopicAsync(string topicName)
        {
            throw new NotImplementedException();
        }

        public Task UnsubscribeAllAsync()
        {
            throw new NotImplementedException();
        }

        public Task UnsubscribeAsync(string topicName, SubscriptionToken subscriptionToken)
        {
            throw new NotImplementedException();
        }

        public Task UpdateSchemaAsync<T>(string topicName)
        {
            throw new NotImplementedException();
        }
    }
}