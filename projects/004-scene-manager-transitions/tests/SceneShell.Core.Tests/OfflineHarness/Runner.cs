// Entry point for offline mode: discovers [Fact]/[Theory] methods by reflection, runs each
// on a fresh instance (as xUnit does), and exits non-zero if any fail.
using System.Reflection;
using Xunit;

namespace SceneShell.Core.Tests.OfflineHarness;

internal static class Runner
{
    private static int Main(string[] args)
    {
        var filter = args.Length > 0 ? args[0] : null;
        var tests = typeof(Runner).Assembly.GetTypes()
            .Where(t => t.IsClass && !t.IsAbstract && t.IsPublic)
            .SelectMany(t => t.GetMethods(BindingFlags.Public | BindingFlags.Instance)
                .Where(m => m.GetCustomAttribute<FactAttribute>() is not null)
                .Select(m => (Type: t, Method: m)))
            .OrderBy(x => x.Type.Name, StringComparer.Ordinal)
            .ThenBy(x => x.Method.Name, StringComparer.Ordinal)
            .ToList();

        var passed = 0;
        var failures = new List<string>();
        foreach (var (type, method) in tests)
        {
            var cases = method.GetCustomAttributes<InlineDataAttribute>().Select(d => d.Data).ToList();
            if (cases.Count == 0)
            {
                cases.Add(Array.Empty<object?>());
            }

            foreach (var data in cases)
            {
                var name = data.Length == 0
                    ? $"{type.Name}.{method.Name}"
                    : $"{type.Name}.{method.Name}({string.Join(", ", data)})";
                if (filter is not null && !name.Contains(filter, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }

                try
                {
                    var instance = Activator.CreateInstance(type);
                    try
                    {
                        method.Invoke(instance, data.Length == 0 ? null : data);
                    }
                    finally
                    {
                        (instance as IDisposable)?.Dispose();
                    }
                    passed++;
                }
                catch (TargetInvocationException ex) when (ex.InnerException is not null)
                {
                    failures.Add($"FAIL {name}\n     {ex.InnerException.GetType().Name}: {ex.InnerException.Message}");
                }
            }
        }

        foreach (var failure in failures)
        {
            Console.WriteLine(failure);
        }
        Console.WriteLine($"Passed: {passed}, Failed: {failures.Count}, Total: {passed + failures.Count}");
        return failures.Count == 0 ? 0 : 1;
    }
}
