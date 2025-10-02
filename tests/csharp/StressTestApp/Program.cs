namespace DataLoomStressTest
{
    internal partial class Program
    {
        static async Task Main(string[] args)
        {
            if (args.Length == 0)
            {
                Console.WriteLine("Usage: dotnet run <test-name>");
                return;
            }

            switch (args[0].ToLowerInvariant())
            {
                case "clients":
                    await RunClientScalingTest();
                    break;
                case "frequency":
                    await RunMessageFrequencyTest();
                    break;
                default:
                    Console.WriteLine("Unknown test. Use 'clients' or 'frequency'.");
                    break;
            }
        }        
    }
}
