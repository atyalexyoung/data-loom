using System.Collections.Concurrent;
using System.Diagnostics;
using DataLoom.SDK.Interfaces;
using DataLoom.SDK.Subscriptions;
using IntegrationTests.Helpers;
using IntegrationTests.Models;

namespace DataLoomStressTest
{
    internal class Program
    {
        private static async Task Main(string[] args)
        {
            const int clientCount = 100; // adjust 100–500
            const string topicName = "stress-test-topic";

            Console.WriteLine($"Starting stress test with {clientCount} clients...");

            var clients = new IMessagingClient[clientCount];
            var subscriptions = new SubscriptionToken[clientCount];

            var roundTripTimes = new ConcurrentBag<long>();
            var failures = new ConcurrentBag<string>();

            // Launch and connect all clients in parallel
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                var client = TestClientFactory.BuildClient($"stress-client-{i}");
                try
                {
                    await client.ConnectAsync();
                    clients[i] = client;
                }
                catch (Exception ex)
                {
                    failures.Add($"Client {i} failed to connect: {ex.Message}");
                }
            }));

            Console.WriteLine("All clients connected.");

            // Register the topic once with the first client
            await clients[0].RegisterTopicAsync<ChatMessage>(topicName);

            // Subscribe all clients to the topic
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                try
                {
                    var token = await clients[i].SubscribeAsync<ChatMessage>(topicName, async msg =>
                    {
                        var rt = (DateTime.UtcNow - msg.Data.Timestamp).TotalMilliseconds;
                        roundTripTimes.Add((long)rt);
                        await Task.CompletedTask;
                    });
                    subscriptions[i] = token;
                }
                catch (Exception ex)
                {
                    failures.Add($"Client {i} failed to subscribe: {ex.Message}");
                }
            }));

            Console.WriteLine("All clients subscribed.");

            // Publish messages from all clients concurrently
            var sw = Stopwatch.StartNew();
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                try
                {
                    var msg = new ChatMessage
                    {
                        Sender = $"stress-client-{i}",
                        Message = $"Hello from client {i}",
                        Timestamp = DateTime.UtcNow
                    };
                    await clients[i].PublishAsync(topicName, msg);
                }
                catch (Exception ex)
                {
                    failures.Add($"Client {i} failed to publish: {ex.Message}");
                }
            }));
            sw.Stop();

            // Wait a short period for messages to be received
            await Task.Delay(2000);

            // Print metrics
            Console.WriteLine("==== Stress Test Metrics ====");
            Console.WriteLine($"Total clients: {clientCount}");
            Console.WriteLine($"Total failures: {failures.Count}");
            if (failures.Count > 0)
                foreach (var f in failures) Console.WriteLine(f);

            if (roundTripTimes.Count > 0)
            {
                var times = roundTripTimes.ToArray();
                Console.WriteLine($"Messages received: {times.Length}");
                Console.WriteLine($"Avg round-trip (ms): {times.Average():F2}");
                Console.WriteLine($"Min round-trip (ms): {times.Min()}");
                Console.WriteLine($"Max round-trip (ms): {times.Max()}");
            }

            Console.WriteLine($"Total publish time (ms): {sw.ElapsedMilliseconds}");

            // Cleanup
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                try
                {
                    await clients[i].UnsubscribeAsync(topicName, subscriptions[i]);
                    await clients[i].DisconnectAsync();
                }
                catch { }
            }));

            Console.WriteLine("Stress test complete.");
        }
    }
}
