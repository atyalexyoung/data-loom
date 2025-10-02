using System.Collections.Concurrent;
using System.Diagnostics;
using System.Globalization;
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
            // Configurable parameters per run
            const int clientCount = 20;   // keep this fixed across runs
            const int msgPerSecond = 100;  // vary this per run!
            const int testDurationSec = 30;
            const string resultsFile = "freq_results.csv";

            var topics = new[] { "chat", "alerts", "metrics", "logs" }; // hardcoded topics

            Console.WriteLine($"Starting stress test with {clientCount} clients, " +
                              $"{msgPerSecond} msgs/sec for {testDurationSec} sec...");

            var clients = new IMessagingClient[clientCount];
            var subscriptions = new List<SubscriptionToken>[clientCount];

            var roundTripTimes = new ConcurrentBag<long>();
            var failures = new ConcurrentBag<string>();

            // 1. Connect all clients
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                var client = TestClientFactory.BuildClient($"client-{i}");
                try
                {
                    await client.ConnectAsync();
                    clients[i] = client;
                    subscriptions[i] = new List<SubscriptionToken>();
                }
                catch (Exception ex)
                {
                    failures.Add($"Client {i} failed to connect: {ex.Message}");
                }
            }));
            Console.WriteLine("All clients connected.");

            // 2. Register all topics (once)
            foreach (var topic in topics)
            {
                await clients[0].RegisterTopicAsync<ChatMessage>(topic);
            }

            // 3. Randomly subscribe clients to subset of topics
            var rand = new Random();
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                try
                {
                    var chosenTopics = topics.Where(_ => rand.NextDouble() < 0.5).ToList();
                    foreach (var topic in chosenTopics)
                    {
                        if (topic == "chat")
                            subscriptions[i].Add(await clients[i].SubscribeAsync<ChatMessage>(topic, async msg =>
                            {
                                RecordRt(msg.Data.Timestamp, roundTripTimes);
                                await Task.CompletedTask;
                            }));
                        else if (topic == "alerts")
                            subscriptions[i].Add(await clients[i].SubscribeAsync<AlertMessage>(topic, async msg =>
                            {
                                RecordRt(msg.Data.Timestamp, roundTripTimes);
                                await Task.CompletedTask;
                            }));
                        else if (topic == "metrics")
                            subscriptions[i].Add(await clients[i].SubscribeAsync<MetricMessage>(topic, async msg =>
                            {
                                RecordRt(msg.Data.Timestamp, roundTripTimes);
                                await Task.CompletedTask;
                            }));
                        else if (topic == "logs")
                            subscriptions[i].Add(await clients[i].SubscribeAsync<LogMessage>(topic, async msg =>
                            {
                                RecordRt(msg.Data.Timestamp, roundTripTimes);
                                await Task.CompletedTask;
                            }));
                    }
                }
                catch (Exception ex)
                {
                    failures.Add($"Client {i} failed to subscribe: {ex.Message}");
                }
            }));
            Console.WriteLine("All clients subscribed.");

            // 4. Start publishers (fixed frequency across testDurationSec)
            var publishTasks = new List<Task>();
            var sw = Stopwatch.StartNew();
            var cts = new CancellationTokenSource(TimeSpan.FromSeconds(testDurationSec));

            // Publisher loop: select random client + random topic each tick
            publishTasks.Add(Task.Run(async () =>
            {
                var intervalMs = 1000.0 / msgPerSecond;
                var nextTick = sw.ElapsedMilliseconds;

                while (!cts.IsCancellationRequested)
                {
                    var clientIdx = rand.Next(clientCount);
                    var topic = topics[rand.Next(topics.Length)];
                    var msg = new ChatMessage
                    {
                        Sender = $"client-{clientIdx}",
                        Message = $"Hello from {clientIdx} at {DateTime.UtcNow:O}",
                        Timestamp = DateTime.UtcNow
                    };

                    try
                    {
                        await clients[clientIdx].PublishAsync(topic, msg);
                    }
                    catch (Exception ex)
                    {
                        failures.Add($"Client {clientIdx} failed to publish: {ex.Message}");
                    }

                    nextTick += (long)intervalMs;
                    var delay = nextTick - sw.ElapsedMilliseconds;
                    if (delay > 0) await Task.Delay((int)delay);
                }
            }));

            await Task.WhenAll(publishTasks);
            sw.Stop();

            // 5. Collect results
            await Task.Delay(2000); // let last messages arrive
            var received = roundTripTimes.Count;
            var avg = received > 0 ? roundTripTimes.Average() : 0;
            var min = received > 0 ? roundTripTimes.Min() : 0;
            var max = received > 0 ? roundTripTimes.Max() : 0;

            Console.WriteLine("==== Frequency Test Metrics ====");
            Console.WriteLine($"Clients: {clientCount}");
            Console.WriteLine($"Topics: {topics.Length}");
            Console.WriteLine($"Msg/s: {msgPerSecond}");
            Console.WriteLine($"Duration (s): {testDurationSec}");
            Console.WriteLine($"Failures: {failures.Count}");
            Console.WriteLine($"Messages received: {received}");
            Console.WriteLine($"Avg round-trip (ms): {avg:F2}");
            Console.WriteLine($"Min round-trip (ms): {min}");
            Console.WriteLine($"Max round-trip (ms): {max}");

            // 6. Write summary to CSV
            WriteMessageFrequencyResultsToCsv(resultsFile, clientCount, topics.Length, msgPerSecond, testDurationSec,
                failures.Count, received, avg, min, max);

            // 7. Cleanup
            await Task.WhenAll(Enumerable.Range(0, clientCount).Select(async i =>
            {
                try
                {
                    foreach (var sub in subscriptions[i])
                        await clients[i].UnsubscribeAsync("unused", sub); // unsub
                    await clients[i].DisconnectAsync();
                }
                catch { }
            }));

            Console.WriteLine("Test complete.");
        }

        private static void RecordRt(DateTime timestamp, ConcurrentBag<long> roundTripTimes)
        {
            var rt = (DateTime.UtcNow - timestamp).TotalMilliseconds;
            roundTripTimes.Add((long)rt);
        }

        private static void WriteMessageFrequencyResultsToCsv(
            string filePath, int clients, int topics, int msgPerSecond, int durationSec,
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
                DateTime.UtcNow.ToString("o", CultureInfo.InvariantCulture),
                clients.ToString(),
                topics.ToString(),
                msgPerSecond.ToString(),
                durationSec.ToString(),
                failures.ToString(),
                received.ToString(),
                avg.ToString("F2", CultureInfo.InvariantCulture),
                min.ToString(),
                max.ToString()
            }));
        }
    }
}