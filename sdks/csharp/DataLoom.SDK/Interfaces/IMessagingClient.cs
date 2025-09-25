using DataLoom.SDK.Exceptions;
using DataLoom.SDK.Models;
using DataLoom.SDK.Subscriptions;

namespace DataLoom.SDK.interfaces
{
    /// <summary>
    /// IMessagingClient is the interface for interacting with a messaging server.
    /// It provides methods to connect, subscribe, publish, and manage topics.
    /// </summary>
    public interface IMessagingClient
    {
        /// <summary>
        /// Connects the client to the server using the configured URL and API key.
        /// </summary>
        /// <remarks>
        /// This method asynchronously opens a WebSocket connection to the server. 
        /// If the connection fails (e.g., due to network issues, invalid URL, or protocol errors), 
        /// an exception such as <see cref="WebSocketException"/> will be thrown.
        /// 
        /// After a successful connection, the receive loop is started in a fire-and-forget task. 
        /// Exceptions thrown inside the receive loop will not propagate to this method. 
        /// Consider monitoring them via logging or events if you need to handle fatal errors during message processing.
        /// </remarks>
        /// <exception cref="WebSocketException">Thrown if the connection to the server fails.</exception>
        /// <exception cref="ArgumentException">Thrown if the server URL is invalid.</exception>
        /// <exception cref="UriFormatException">Thrown if the server URL is not a valid URI.</exception>
        /// <exception cref="InvalidOperationException">Thrown if the WebSocket is already connected or if options are invalid.</exception>
        Task ConnectAsync();

        /// <summary>
        /// Subscribes to a topic with the specified name and handles incoming messages using the provided callback.
        /// </summary>
        /// <typeparam name="T">The type of the message data for the topic.</typeparam>
        /// <param name="topicName">The name of the topic to subscribe to.</param>
        /// <param name="onMessageReceivedCallback">Async callback invoked when a message is received on the topic.</param>
        /// <returns>A task that completes with a <see cref="SubscriptionToken"/> for managing the subscription.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the subscription request.</exception>
        Task<SubscriptionToken> SubscribeAsync<T>(string topicName, Func<WebSocketMessage<T>, Task> onMessageReceivedCallback);

        /// <summary>
        /// Publishes a message to a topic.
        /// </summary>
        /// <typeparam name="T">The type of the message data.</typeparam>
        /// <param name="topicName">The name of the topic to publish to.</param>
        /// <param name="value">The message data to publish.</param>
        /// <returns>A task representing the asynchronous publish operation.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the publish request.</exception>
        Task PublishAsync<T>(string topicName, T value);

        /// <summary>
        /// Unsubscribes from a topic for the given subscription token.
        /// </summary>
        /// <param name="topicName">The name of the topic to unsubscribe from.</param>
        /// <param name="subscriptionToken">The subscription token associated with the subscription.</param>
        /// <returns>A task representing the asynchronous unsubscribe operation.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="SubscriptionFailedException">Thrown if the subscription could not be removed locally.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the unsubscribe request.</exception>
        Task UnsubscribeAsync(string topicName, SubscriptionToken subscriptionToken);

        /// <summary>
        /// Unsubscribes from all topics for this client.
        /// </summary>
        /// <returns>A task representing the asynchronous operation.</returns>
        /// <exception cref="ServerException">Thrown if the server cannot process the unsubscribe request.</exception>
        Task UnsubscribeAllAsync();

        /// <summary>
        /// Retrieves the latest value for a topic.
        /// </summary>
        /// <typeparam name="T">The type of the value to retrieve.</typeparam>
        /// <param name="topicName">The name of the topic.</param>
        /// <returns>A task that completes with the value, or null if no value exists.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the get request.</exception>
        Task<T?> GetAsync<T>(string topicName);

        /// <summary>
        /// Registers a new topic with the specified name and schema type.
        /// </summary>
        /// <typeparam name="T">The type representing the structure of the topic.</typeparam>
        /// <param name="topicName">The name of the topic to register.</param>
        /// <returns>A task representing the asynchronous operation.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the register request.</exception>
        Task RegisterTopicAsync<T>(string topicName);

        /// <summary>
        /// Unregisters a topic with the specified name.
        /// This stops all publishing and subscriptions for the topic across all clients.
        /// </summary>
        /// <param name="topicName">The name of the topic to unregister.</param>
        /// <returns>A task representing the asynchronous operation.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the unregister request.</exception>
        Task UnregisterTopicAsync(string topicName);

        /// <summary>
        /// Retrieves a list of all currently registered topics on the server.
        /// </summary>
        /// <returns>A task that completes with an enumerable of topic names.</returns>
        /// <exception cref="ServerException">Thrown if the server cannot process the list request.</exception>
        Task<IEnumerable<string>> ListTopicsAsync();

        /// <summary>
        /// Updates the schema of an existing topic to a new type.
        /// </summary>
        /// <typeparam name="T">The type representing the new schema for the topic.</typeparam>
        /// <param name="topicName">The name of the topic to update.</param>
        /// <returns>A task representing the asynchronous operation.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the update request.</exception>
        Task UpdateSchemaAsync<T>(string topicName);

        /// <summary>
        /// Publishes a message to a topic without persisting it to storage.
        /// </summary>
        /// <typeparam name="T">The type of the message data.</typeparam>
        /// <param name="topicName">The name of the topic to publish to.</param>
        /// <param name="value">The message data to publish.</param>
        /// <returns>A task representing the asynchronous operation.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task SendWithoutSaveAsync<T>(string topicName, T value);
    }
}