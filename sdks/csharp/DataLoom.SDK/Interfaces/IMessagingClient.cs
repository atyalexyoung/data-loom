using DataLoom.SDK.Exceptions;
using DataLoom.SDK.Models;
using DataLoom.SDK.Subscriptions;

namespace DataLoom.SDK.interfaces
{
    /// <summary>
    /// IMessagingClient is the interface to interact with messaging server. It provides methods to
    /// initialize, cleanup, and perform actions.
    /// </summary>
    public interface IMessagingClient
    {
        /// <summary>
        /// SubscribeAsync will subscribe to the topic with the name provided and use the callback provided as a handler
        /// for the values that come in.
        /// </summary>
        /// <typeparam name="T">The type of the topic that is being subscribed to.</typeparam>
        /// <param name="topicName">The name of the topic that is being subscribed to.</param>
        /// <param name="onMessageReceivedCallback">Callback for the subscription to act as async handler. </param>
        /// <returns>Task.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task<SubscriptionToken> SubscribeAsync<T>(string topicName, Func<WebSocketMessage<T>, Task> onMessageReceivedCallback);

        /// <summary>
        /// PublishAsync will publish to a topic with the name provided with the value that is provided.
        /// </summary>
        /// <typeparam name="T">The type of the message that is being published.</param>
        /// <param name="topicName">The name of the topic that the message is being published on.</param>
        /// <param name="value">The value of the message to be published.</param>
        /// <returns>Task.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task PublishAsync<T>(string topicName, T value);

        /// <summary>
        /// UnsubscribeAsync will usubscribe from the topic of the name provided.
        /// </summary>
        /// <param name="topicName">The name of the topic to unsubscribe from.</param>
        /// <param name="subscriptionToken">The token of the subscription to unsubscribe from </param>
        /// <returns>Task</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task UnsubscribeAsync(string topicName, SubscriptionToken subscriptionToken);

        /// <summary>
        /// UnsubscribeAllAsync will unsubscribe from all topics for this client.
        /// </summary>
        /// <returns>Task</returns>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task UnsubscribeAllAsync();

        /// <summary>
        /// GetAsync will retrieve a value for the topic of the name provided.
        /// </summary>
        /// <typeparam name="T">The type of the value that will be retreived.</typeparam>
        /// <param name="topicName">The name of the topic to retreive the value of.</param>
        /// <returns>A Task with value of type T provided.</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task<T?> GetAsync<T>(string topicName);

        /// <summary>
        /// RegisterTopicAsync will register a topic with the name provided and structure of the type
        /// that is specified.
        /// </summary>
        /// <typeparam name="T">The type of the topic.</typeparam>
        /// <param name="topicName">The name of the topic.</param>
        /// <returns>Task</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task RegisterTopicAsync<T>(string topicName);

        /// <summary>
        /// UnregisterTopicAsync will unregister a topic of the name provided from the server. This will
        /// stop all publishing and subscriptions to this particular topic for ALL clients.
        /// </summary>
        /// <param name="topicName">The name of the topic to unregister.</param>
        /// <returns>Task</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task UnregisterTopicAsync(string topicName);

        /// <summary>
        /// ListTopicsAsync will get the names of all the current topics registered on the server.
        /// </summary>
        /// <returns>Task with IEnumerable of strings of the topic names.</returns>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task<IEnumerable<string>> ListTopicsAsync();

        /// <summary>
        /// UpdateSchemaAsync will update the structure of the topic of the name provided to the type
        /// that is specified.
        /// </summary>
        /// <typeparam name="T">The new type for the structure of the schema.</typeparam>
        /// <param name="topicName">The name of the topic to update the schema for.</param>
        /// <returns>Task</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot processf the request.</exception>
        Task UpdateSchemaAsync<T>(string topicName);

        /// <summary>
        /// SendWithoutSaveAsync will publish on a topic of the name provided with a message, and will
        /// not persist the message to storage.
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="topicName"></param>
        /// <param name="value"></param>
        /// <returns>Task</returns>
        /// <exception cref="ArgumentException">Thrown if the topic name is invalid.</exception>
        /// <exception cref="ServerException">Thrown if the server cannot process the request.</exception>
        Task SendWithoutSaveAsync<T>(string topicName, T value);
    }
}