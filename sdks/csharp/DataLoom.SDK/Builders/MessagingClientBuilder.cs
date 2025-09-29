using DataLoom.SDK.Clients;
using DataLoom.SDK.Interfaces;
using DataLoom.SDK.Models;

namespace DataLoom.SDK.Builders
{
    /// <summary>
    /// MessagingClientBuilder is the builder class for a MessagingClient and provides
    /// a fluent, builder-pattern style way to create and configure a messaging client.
    /// </summary>
    public class MessagingClientBuilder
    {
        private MessagingClientOptions _options;

        /// <summary>
        /// Creates MessagingClientBuilder with default options. 
        /// Currently, the API key remains null and the server url is default as a blank string.
        /// </summary>
        public MessagingClientBuilder()
        {
            _options = new MessagingClientOptions();
        }

        /// <summary>
        /// Adds API Key to configuration of messaging client.
        /// </summary>
        /// <param name="apiKey">The API key to use when connecting to server.</param>
        /// <returns>MessagingClientBuilder with new options configuration.</returns>
        public MessagingClientBuilder WithApiKey(string apiKey)
        {
            _options.ApiKey = apiKey;
            return this;
        }

        /// <summary>
        /// Adds URL of the server to connect to.
        /// </summary>
        /// <param name="url">The URL of the server to connect to.</param>
        /// <returns>MessagingClientBuilder with new options configuration.</returns>
        public MessagingClientBuilder WithServerUrl(string url)
        {
            _options.ServerUrl = url;
            return this;
        }

        /// <summary>
        /// Sets the option for server to send acknowledgmenets of actions back to client or not.
        /// </summary>
        /// <param name="isServerAckEnabled">Boolean of if the server should send ACK's back to the client or not.</param>
        /// <returns>MessagingClientBuilder with new options configuration.</returns>
        public MessagingClientBuilder WithAckFromServer(bool isServerAckEnabled)
        {
            _options.IsServerAckEnabled = isServerAckEnabled;
            return this;
        }

        /// <summary>
        /// Sets the number of reconnections retries the client will do if disconnected from the  server.
        /// </summary>
        /// <param name="numRetries">The number of retries to perform when reconnecting to server.</param>
        /// <returns>MessagingClientBuilder with new options configuration.</returns>
        public MessagingClientBuilder WithReconnectRetried(int numRetries)
        {
            _options.NumRetries = numRetries;
            return this;
        }

        /// <summary>
        /// Sets the ID of the client which can be seen as part of the data payload on other clients or the server.
        /// </summary>
        /// <param name="clientId">The ID for the client being configured.</param>
        /// <returns>MessagingClientBuilder with new options configuration.</returns>
        public MessagingClientBuilder WithClientId(string clientId)
        {
            _options.ClientId = clientId;
            return this;
        }

        /// <summary>
        /// Will build and return the new MessagingClient with the options configured in the builder.
        /// </summary>
        /// <returns>IMessagingClient implementation.</returns>
        /// <exception cref="InvalidOperationException">Exception thrown when Server URL or API key is not set properly.</exception>
        public IMessagingClient Build()
        {
            if (string.IsNullOrWhiteSpace(_options.ServerUrl))
            {
                throw new InvalidOperationException("Server URL null, empty or whitespace.");
            }
            var client = new MessagingClient(_options);
            return client;
        }
    }
}