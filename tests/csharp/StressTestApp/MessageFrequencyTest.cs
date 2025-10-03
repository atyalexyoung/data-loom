
using DataLoom.SDK.Interfaces;
using DataLoom.SDK.Subscriptions;
using IntegrationTests.Helpers;
using IntegrationTests.Models;

namespace DataLoomStressTest
{
    internal partial class Program
    {
        private static async Task RunMessageFrequencyTest()
        {
            const int clientCount = 20;
            const int msgPerSecond = 700; // moderate rate
            const int testDurationSec = 30;
            const string resultsFile = "freq_results.csv";

            var topics = new[] { "chat", "alerts", "metrics", "logs" };
            Console.WriteLine($"Starting stress test with {clientCount} clients, {msgPerSecond} msgs/sec for {testDurationSec} sec...");

            var clients = new IMessagingClient[clientCount];
            var subscriptions = new Dictionary<string, SubscriptionToken>[clientCount];
            var roundTripTimes = new List<long>();
            var failures = new List<string>();
            var rand = new Random();

            // 1. Connect clients
            for (int i = 0; i < clientCount; i++)
            {
                var client = TestClientFactory.BuildClient($"client-{i}");
                try
                {
                    await client.ConnectAsync();
                    clients[i] = client;
                    subscriptions[i] = new Dictionary<string, SubscriptionToken>();
                }
                catch (Exception ex)
                {
                    failures.Add($"Client {i} failed to connect: {ex.Message}");
                }
            }

            // 2. Register topics once
            await clients[0].RegisterTopicAsync<ChatMessage>("chat");
            await clients[0].RegisterTopicAsync<AlertMessage>("alerts");
            await clients[0].RegisterTopicAsync<MetricMessage>("metrics");
            await clients[0].RegisterTopicAsync<LogMessage>("logs");

            // 3. Subscribe clients (sequentially, low memory)
            for (int i = 0; i < clientCount; i++)
            {
                foreach (var topic in topics)
                {
                    SubscriptionToken token = topic switch
                    {
                        "chat" => await clients[i].SubscribeAsync<ChatMessage>(topic, msg => { RecordRt(msg.Data.Timestamp, roundTripTimes); return Task.CompletedTask; }),
                        "alerts" => await clients[i].SubscribeAsync<AlertMessage>(topic, msg => { RecordRt(msg.Data.Timestamp, roundTripTimes); return Task.CompletedTask; }),
                        "metrics" => await clients[i].SubscribeAsync<MetricMessage>(topic, msg => { RecordRt(msg.Data.Timestamp, roundTripTimes); return Task.CompletedTask; }),
                        "logs" => await clients[i].SubscribeAsync<LogMessage>(topic, msg => { RecordRt(msg.Data.Timestamp, roundTripTimes); return Task.CompletedTask; }),
                        _ => null
                    };
                    if (token != null) subscriptions[i][topic] = token;
                }
            }

            // 4. Start one async loop per client for publishing
            var cts = new CancellationTokenSource(TimeSpan.FromSeconds(testDurationSec));
            var publisherTasks = new List<Task>();

            foreach (var clientIdx in Enumerable.Range(0, clientCount))
            {
                publisherTasks.Add(Task.Run(async () =>
                {
                    var intervalMs = 1000.0 / msgPerSecond;
                    while (!cts.Token.IsCancellationRequested)
                    {
                        string topic = topics[rand.Next(topics.Length)];
                        object msg = topic switch
                        {
                            "chat" => new ChatMessage { Sender = $"client-{clientIdx}", Message = "Hello", Timestamp = DateTime.UtcNow },
                            "alerts" => new AlertMessage { Level = $"client-{clientIdx}", Message = "Alert", Timestamp = DateTime.UtcNow },
                            "metrics" => new MetricMessage { Name = $"client-{clientIdx}", Value = rand.NextDouble(), Timestamp = DateTime.UtcNow },
                            "logs" => new LogMessage { Source = $"client-{clientIdx}", Message = "Log", Timestamp = DateTime.UtcNow },
                            _ => null
                        };

                        if (msg != null)
                        {
                            try
                            {
                                _ = topic switch
                                {
                                    "chat" => clients[clientIdx].PublishAsync(topic, (ChatMessage)msg),
                                    "alerts" => clients[clientIdx].PublishAsync(topic, (AlertMessage)msg),
                                    "metrics" => clients[clientIdx].PublishAsync(topic, (MetricMessage)msg),
                                    "logs" => clients[clientIdx].PublishAsync(topic, (LogMessage)msg),
                                    _ => Task.CompletedTask
                                };
                            }
                            catch (Exception ex)
                            {
                                failures.Add($"Client {clientIdx} failed to publish: {ex.Message}");
                            }
                        }

                        try
                        {
                            await Task.Delay((int)intervalMs, cts.Token);
                        }
                        catch (TaskCanceledException)
                        {
                            // Test finished, exit loop
                            break;
                        }
                    }
                }));
            }

            // 5. Wait for test duration
            await Task.WhenAll(publisherTasks);

            // 6. Collect metrics
            var received = roundTripTimes.Count;
            var avg = received > 0 ? roundTripTimes.Average() : 0;
            var min = received > 0 ? roundTripTimes.Min() : 0;
            var max = received > 0 ? roundTripTimes.Max() : 0;

            WriteMessageFrequencyResultsToCsv(
                resultsFile,
                clientCount,
                topics.Length,
                msgPerSecond,
                testDurationSec,
                failures.Count,
                received,
                avg,
                min,
                max);

            Console.WriteLine($"Failures: {failures.Count}, Messages received: {received}, Avg RT: {avg:F2} ms, Min: {min}, Max: {max}");

            // 7. Cleanup
            for (int i = 0; i < clientCount; i++)
            {
                foreach (var kvp in subscriptions[i])
                    await clients[i].UnsubscribeAsync(kvp.Key, kvp.Value);
                await clients[i].DisconnectAsync();
            }

            Console.WriteLine("Test complete.");
            Console.WriteLine($"Failures: {failures.Count}, Messages received: {received}, Avg RT: {avg:F2} ms, Min: {min}, Max: {max}");
        }

        private static void WriteMessageFrequencyResultsToCsv(
            string filePath, int clients, int topicsCount, int msgPerSecond, int durationSec,
            int failures, int received, double avg, long min, long max)
        {
            var newFile = !File.Exists(filePath);
            using var writer = new StreamWriter(filePath, append: true);
            if (newFile)
            {
                writer.WriteLine("Timestamp,Clients,Topics,MsgPerSec,DurationSec,Failures,MessagesReceived,AvgLatencyMs,MinLatencyMs,MaxLatencyMs");
            }
            writer.WriteLine(string.Join(",", new[]
            {
                DateTime.UtcNow.ToString("o"),
                clients.ToString(),
                topicsCount.ToString(),
                msgPerSecond.ToString(),
                durationSec.ToString(),
                failures.ToString(),
                received.ToString(),
                avg.ToString("F2"),
                min.ToString(),
                max.ToString()
            }));
        }


        private static void RecordRt(DateTime timestamp, List<long> roundTripTimes)
        {
            roundTripTimes.Add((long)(DateTime.UtcNow - timestamp).TotalMilliseconds);
        }
    }
}
