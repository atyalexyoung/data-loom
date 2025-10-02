using System.Collections.Concurrent;
using System.Diagnostics;
using System.Globalization;
using System.Text;
using DataLoom.SDK.Interfaces;
using DataLoom.SDK.Subscriptions;
using IntegrationTests.Helpers;
using IntegrationTests.Models;

namespace DataLoomStressTest
{
    internal partial class Program
    {
        private static async Task RunClientScalingTest()
        {
            const int clientCount = 75; // vary this between runs
            const string topicName = "stress-test-topic";
            const int publishBatchSize = 50; // how many clients publish concurrently
            const string resultsFile = "stress_results.csv"; // CSV log file

            Console.WriteLine($"Starting stress test with {clientCount} clients...");

            var clients = new IMessagingClient[clientCount];
            var subscriptions = new SubscriptionToken[clientCount];

            var roundTripTimes = new ConcurrentBag<long>();
            var failures = new ConcurrentBag<string>();

            // 1. Connect all clients
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

            // 2. Register topic once
            await clients[0].RegisterTopicAsync<ChatMessage>(topicName);

            // 3. Subscribe all clients
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                try
                {
                    var localClient = clients[i];
                    subscriptions[i] = await localClient.SubscribeAsync<ChatMessage>(topicName, async msg =>
                    {
                        var rt = (DateTime.UtcNow - msg.Data.Timestamp).TotalMilliseconds;
                        roundTripTimes.Add((long)rt);
                        await Task.CompletedTask; // fire-and-forget
                    });
                }
                catch (Exception ex)
                {
                    failures.Add($"Client {i} failed to subscribe: {ex.Message}");
                }
            }));

            Console.WriteLine("All clients subscribed.");

            // 4. Publish messages in batches
            var sw = Stopwatch.StartNew();
            for (int batchStart = 0; batchStart < clientCount; batchStart += publishBatchSize)
            {
                var batchTasks = Enumerable.Range(batchStart, Math.Min(publishBatchSize, clientCount - batchStart)).Select(async i =>
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
                });

                await Task.WhenAll(batchTasks);
            }
            sw.Stop();

            // 5. Wait for messages to arrive
            await Task.Delay(2000);

            // 6. Aggregate metrics
            var messagesReceived = roundTripTimes.Count;
            var avg = roundTripTimes.Count > 0 ? roundTripTimes.Average() : 0;
            var min = roundTripTimes.Count > 0 ? roundTripTimes.Min() : 0;
            var max = roundTripTimes.Count > 0 ? roundTripTimes.Max() : 0;

            Console.WriteLine("==== Stress Test Metrics ====");
            Console.WriteLine($"Total clients: {clientCount}");
            Console.WriteLine($"Total failures: {failures.Count}");
            Console.WriteLine($"Messages received: {messagesReceived}");
            Console.WriteLine($"Avg round-trip (ms): {avg:F2}");
            Console.WriteLine($"Min round-trip (ms): {min}");
            Console.WriteLine($"Max round-trip (ms): {max}");
            Console.WriteLine($"Total publish time (ms): {sw.ElapsedMilliseconds}");

            // 7. Write metrics to CSV
            WriteClientScalingResultsToCsv(resultsFile, clientCount, failures.Count, messagesReceived, avg, min, max, sw.ElapsedMilliseconds);

            // 8. Cleanup
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

        private static void WriteClientScalingResultsToCsv(
            string filePath, int clients, int failures, int received,
            double avg, long min, long max, long publishTimeMs)
        {
            var newFile = !File.Exists(filePath);

            var sb = new StringBuilder();
            if (newFile)
            {
                sb.AppendLine("Timestamp,Clients,Failures,MessagesReceived,AvgLatencyMs,MinLatencyMs,MaxLatencyMs,PublishTimeMs");
            }

            sb.AppendLine(string.Join(",", new[]
            {
                DateTime.UtcNow.ToString("o", CultureInfo.InvariantCulture),
                clients.ToString(),
                failures.ToString(),
                received.ToString(),
                avg.ToString("F2", CultureInfo.InvariantCulture),
                min.ToString(),
                max.ToString(),
                publishTimeMs.ToString()
            }));

            File.AppendAllText(filePath, sb.ToString());
        }
    }
}