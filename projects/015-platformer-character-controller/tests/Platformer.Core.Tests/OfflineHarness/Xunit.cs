// Offline stand-in for the subset of xUnit this suite uses. Compiled only when building
// with -p:OfflineHarness=true; a normal build references the real xUnit packages instead.
// It mirrors xUnit's names and semantics so the test files compile unchanged either way.
using System.Collections;

namespace Xunit;

[AttributeUsage(AttributeTargets.Method)]
public class FactAttribute : Attribute { }

[AttributeUsage(AttributeTargets.Method)]
public sealed class TheoryAttribute : FactAttribute { }

[AttributeUsage(AttributeTargets.Method, AllowMultiple = true)]
public sealed class InlineDataAttribute : Attribute
{
    public InlineDataAttribute(params object?[] data) => Data = data;

    public object?[] Data { get; }
}

public sealed class AssertionException : Exception
{
    public AssertionException(string message) : base(message) { }
}

public static class Assert
{
    public static void True(bool condition, string? userMessage = null)
    {
        if (!condition)
        {
            throw new AssertionException(userMessage ?? "Expected true, got false.");
        }
    }

    public static void False(bool condition, string? userMessage = null)
    {
        if (condition)
        {
            throw new AssertionException(userMessage ?? "Expected false, got true.");
        }
    }

    public static void Equal<T>(T expected, T actual)
    {
        if (expected is IEnumerable e && actual is IEnumerable a && expected is not string)
        {
            SequenceEqual(e.Cast<object?>(), a.Cast<object?>());
            return;
        }
        if (!EqualityComparer<T>.Default.Equals(expected, actual))
        {
            throw new AssertionException($"Expected: {Show(expected)}\nActual:   {Show(actual)}");
        }
    }

    public static void Equal<T>(IEnumerable<T> expected, IEnumerable<T> actual) =>
        SequenceEqual(expected.Cast<object?>(), actual.Cast<object?>());

    public static void Equal(double expected, double actual, int precision)
    {
        if (Math.Round(expected, precision) != Math.Round(actual, precision))
        {
            throw new AssertionException($"Expected: {expected} (rounded to {precision} places)\nActual:   {actual}");
        }
    }

    public static void NotEqual<T>(T expected, T actual)
    {
        if (EqualityComparer<T>.Default.Equals(expected, actual))
        {
            throw new AssertionException($"Expected a value other than {Show(expected)}.");
        }
    }

    public static void Null(object? value)
    {
        if (value is not null)
        {
            throw new AssertionException($"Expected null, got {Show(value)}.");
        }
    }

    public static void NotNull(object? value)
    {
        if (value is null)
        {
            throw new AssertionException("Expected a non-null value.");
        }
    }

    public static void Same(object? expected, object? actual)
    {
        if (!ReferenceEquals(expected, actual))
        {
            throw new AssertionException("Expected the same instance.");
        }
    }

    public static void NotSame(object? expected, object? actual)
    {
        if (ReferenceEquals(expected, actual))
        {
            throw new AssertionException("Expected different instances.");
        }
    }

    public static void Empty(IEnumerable collection)
    {
        if (collection.Cast<object?>().Any())
        {
            throw new AssertionException("Expected an empty collection.");
        }
    }

    public static void NotEmpty(IEnumerable collection)
    {
        if (!collection.Cast<object?>().Any())
        {
            throw new AssertionException("Expected a non-empty collection.");
        }
    }

    public static void All<T>(IEnumerable<T> collection, Action<T> action)
    {
        var index = 0;
        foreach (var item in collection)
        {
            try
            {
                action(item);
            }
            catch (AssertionException ex)
            {
                throw new AssertionException($"Item {index} failed: {ex.Message}");
            }
            index++;
        }
    }

    public static T Single<T>(IEnumerable<T> collection)
    {
        var items = collection.ToList();
        if (items.Count != 1)
        {
            throw new AssertionException($"Expected exactly one item, found {items.Count}.");
        }
        return items[0];
    }

    public static void Contains<T>(T expected, IEnumerable<T> collection)
    {
        if (!collection.Contains(expected))
        {
            throw new AssertionException($"Expected the collection to contain {Show(expected)}.");
        }
    }

    public static void Contains<T>(IEnumerable<T> collection, Predicate<T> filter)
    {
        if (!collection.Any(item => filter(item)))
        {
            throw new AssertionException("Expected the collection to contain a matching item.");
        }
    }

    public static void DoesNotContain<T>(T expected, IEnumerable<T> collection)
    {
        if (collection.Contains(expected))
        {
            throw new AssertionException($"Expected the collection not to contain {Show(expected)}.");
        }
    }

    public static void Contains(string expectedSubstring, string? actualString)
    {
        if (actualString is null || !actualString.Contains(expectedSubstring, StringComparison.Ordinal))
        {
            throw new AssertionException($"Expected {Show(actualString)} to contain {Show(expectedSubstring)}.");
        }
    }

    public static void InRange<T>(T actual, T low, T high) where T : IComparable<T>
    {
        if (actual.CompareTo(low) < 0 || actual.CompareTo(high) > 0)
        {
            throw new AssertionException($"Expected {actual} to be in [{low}, {high}].");
        }
    }

    public static T IsType<T>(object? value)
    {
        if (value is null || value.GetType() != typeof(T))
        {
            throw new AssertionException($"Expected type {typeof(T).Name}, got {value?.GetType().Name ?? "null"}.");
        }
        return (T)value;
    }

    public static T Throws<T>(Action testCode) where T : Exception
    {
        try
        {
            testCode();
        }
        catch (Exception ex) when (ex.GetType() == typeof(T))
        {
            return (T)ex;
        }
        catch (Exception ex)
        {
            throw new AssertionException($"Expected {typeof(T).Name}, got {ex.GetType().Name}: {ex.Message}");
        }
        throw new AssertionException($"Expected {typeof(T).Name}, but nothing was thrown.");
    }

    private static void SequenceEqual(IEnumerable<object?> expected, IEnumerable<object?> actual)
    {
        var e = expected.ToList();
        var a = actual.ToList();
        if (e.Count != a.Count || !e.SequenceEqual(a))
        {
            throw new AssertionException($"Expected: [{string.Join(", ", e.Select(Show))}]\nActual:   [{string.Join(", ", a.Select(Show))}]");
        }
    }

    private static string Show(object? value) => value switch
    {
        null => "null",
        string s => $"\"{s}\"",
        _ => value.ToString() ?? "?",
    };
}
