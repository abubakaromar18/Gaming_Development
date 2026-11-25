using Microsoft.Xna.Framework;
using Microsoft.Xna.Framework.Graphics;
using Platformer.Core.Rendering;

namespace Platformer.Desktop;

/// <summary>
/// <see cref="ICanvas"/> on top of a SpriteBatch: every rectangle is the same 1×1 white
/// texture, scaled and tinted. Must be used between SpriteBatch.Begin and End.
/// </summary>
public sealed class SpriteBatchCanvas : ICanvas
{
    private readonly SpriteBatch _batch;
    private readonly Texture2D _pixel;

    public SpriteBatchCanvas(SpriteBatch batch, Texture2D pixel, int width, int height)
    {
        _batch = batch ?? throw new ArgumentNullException(nameof(batch));
        _pixel = pixel ?? throw new ArgumentNullException(nameof(pixel));
        Width = width;
        Height = height;
    }

    public int Width { get; }
    public int Height { get; }

    public void FillRect(float x, float y, float width, float height, Rgba color)
    {
        if (width <= 0f || height <= 0f || color.A == 0)
        {
            return;
        }
        _batch.Draw(
            _pixel,
            new Vector2(x, y),
            sourceRectangle: null,
            new Color(color.R, color.G, color.B, color.A),
            rotation: 0f,
            origin: Vector2.Zero,
            scale: new Vector2(width, height),
            SpriteEffects.None,
            layerDepth: 0f);
    }
}
