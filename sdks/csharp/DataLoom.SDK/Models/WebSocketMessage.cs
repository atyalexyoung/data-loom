using System.Text.Json.Serialization;
using DataLoom.SDK.Builders;
namespace DataLoom.SDK.Models
{
	public class WebSocketMessage<T>
	{
		/// <summary>
		/// Unique identifier for the message. 
		/// Used by SDK to match requests with responses from the server.
		/// </summary>
		[JsonPropertyName("id")]
		public required string MessageId { get; set; }

		/// <summary>
		/// The Client Id of who sent the message. When publishing, 
		/// this will be the ClientId that is set when configuring
		///  with <see cref="MessagingClientBuilder"/>
		/// </summary>
		[JsonPropertyName("senderId")]
		public string? SenderId { get; set; }

		/// <summary>
		/// The action that is being taken with message. 
		/// (e.g. "publish", "subscribe", etc.)
		/// </summary>
		[JsonPropertyName("action")]
		public required string Action { get; set; }

		/// <summary>
		/// The name of the topic to the message is about (if applicable).
		/// </summary>
		[JsonPropertyName("topic")]
		public string? Topic { get; set; }

		/// <summary>
		/// The "payload" or data of the message.
		/// </summary>
		[JsonPropertyName("data")]
		public T? Data { get; set; }

		/// <summary>
		/// Boolean if the sender requires and acknowledgement from the server.
		/// </summary>
		[JsonPropertyName("requireAck")]
		public bool RequireAck { get; set; }
	}
}