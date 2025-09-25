using System.Text.Json.Serialization;

namespace DataLoom.SDK.Models
{
	public class WebSocketMessage<T>
	{
		[JsonPropertyName("id")]
		public required string Id { get; set; }

		[JsonPropertyName("action")]
		public required string Action { get; set; }

		[JsonPropertyName("topic")]
		public string? Topic { get; set; }

		[JsonPropertyName("data")]
		public T? Data { get; set; }

		[JsonPropertyName("requireAck")]
		public bool RequireAck { get; set; }
	}
}