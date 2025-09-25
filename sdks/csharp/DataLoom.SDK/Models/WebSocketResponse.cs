using System.Text.Json;
using System.Text.Json.Serialization;

namespace DataLoom.SDK.Models
{
    public class WebSocketResponse
    {
        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        [JsonPropertyName("type")]
        public string Type { get; set; } = string.Empty;

        [JsonPropertyName("code")]
        public int Code { get; set; }

        [JsonPropertyName("message")]
        public string? Message { get; set; }

        [JsonPropertyName("data")]
        public JsonElement? Data { get; set; }
    }

    public class WebSocketResponse<T>
    {
        private readonly WebSocketResponse _base;

        public WebSocketResponse(WebSocketResponse baseResponse)
        {
            _base = baseResponse;
        }

        public string Id => _base.Id;
        public string Type => _base.Type;
        public int Code => _base.Code;
        public string? Message => _base.Message;

        public T? Data => _base.Data.HasValue ? _base.Data.Value.Deserialize<T>() : default;
    }
}