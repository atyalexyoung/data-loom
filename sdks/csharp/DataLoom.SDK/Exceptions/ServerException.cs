namespace DataLoom.SDK.exceptions
{
    /// <summary>
    /// Exception thrown when the server cannot process a request.
    /// </summary>
    public class ServerException : Exception
    {
        /// <summary>
        /// Status code returned by the server.
        /// </summary>
        public int StatusCode { get; }

        public ServerException(int statusCode, string message) : base(message)
        {
            StatusCode = statusCode;
        }

        public ServerException(int statusCode, string message, Exception inner) : base(message, inner)
        {
            StatusCode = statusCode;
        }
    }
}