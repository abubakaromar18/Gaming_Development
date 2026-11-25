using Microsoft.Xna.Framework;
using Microsoft.Xna.Framework.Graphics;
using Microsoft.Xna.Framework.Input;
using Platformer.Core.Controller;
using Platformer.Core.Game;
using Platformer.Core.Input;
using Platformer.Core.Randomness;
using Platformer.Core.Rendering;
using Platformer.Core.World;

namespace Platformer.Desktop;

/// <summary>
/// The MonoGame host. It owns the window and GPU resources and nothing else: each frame it
/// polls devices, hands (frame time, input) to the core session, which runs its own fixed
/// 120 Hz steps, and draws the session into a map-sized render target that is scaled up
/// with point sampling and letterboxed to the window.
/// </summary>
public sealed class PlatformerGame : Microsoft.Xna.Framework.Game
{
    private const int WindowScale = 2;

    private readonly PlatformerSession _session;
    private readonly DeviceInput _input = new();
    private readonly int _width;
    private readonly int _height;

    private SpriteBatch? _spriteBatch;
    private Texture2D? _pixel;
    private RenderTarget2D? _frame;
    private SpriteBatchCanvas? _canvas;
    private bool _debug;

    public PlatformerGame(ulong seed)
    {
        var level = DemoLevel.Load();
        _session = new PlatformerSession(level, ControllerTuning.Default, new Pcg32(seed));
        _width = level.Map.PixelWidth;
        _height = level.Map.PixelHeight;

        // The manager registers itself with the Game; its preferred sizes are applied
        // when MonoGame creates the graphics device during Initialize.
        _ = new GraphicsDeviceManager(this)
        {
            PreferredBackBufferWidth = _width * WindowScale,
            PreferredBackBufferHeight = _height * WindowScale,
            SynchronizeWithVerticalRetrace = true,
        };

        // The session runs its own fixed-step accumulator, so MonoGame should hand us the
        // real frame time rather than impose a second fixed step.
        IsFixedTimeStep = false;
        IsMouseVisible = true;
        Window.AllowUserResizing = true;
        Window.Title = "Platformer Character Controller";
    }

    protected override void LoadContent()
    {
        _spriteBatch = new SpriteBatch(GraphicsDevice);
        _pixel = new Texture2D(GraphicsDevice, 1, 1);
        _pixel.SetData(new[] { Color.White });
        _frame = new RenderTarget2D(GraphicsDevice, _width, _height);
        _canvas = new SpriteBatchCanvas(_spriteBatch, _pixel, _width, _height);
    }

    protected override void Update(GameTime gameTime)
    {
        var frameSeconds = (float)gameTime.ElapsedGameTime.TotalSeconds;
        if (!IsActive)
        {
            // Unfocused: drop held keys and feed neutral input so nothing sticks.
            _input.Reset();
            _session.Update(frameSeconds, ControllerInput.None);
            base.Update(gameTime);
            return;
        }

        var (input, commands) = _input.Poll(Keyboard.GetState(), GamePad.GetState(PlayerIndex.One));
        if (commands.Quit)
        {
            Exit();
        }
        if (commands.ToggledAssist != AssistFeature.None)
        {
            _session.ToggleAssist(commands.ToggledAssist);
        }
        if (commands.ToggleDebug)
        {
            _debug = !_debug;
        }
        if (commands.Respawn)
        {
            _session.Respawn();
        }

        _session.Update(frameSeconds, input);
        base.Update(gameTime);
    }

    protected override void Draw(GameTime gameTime)
    {
        var batch = _spriteBatch!;

        // 1. Draw the map-sized logical frame. Colors use straight alpha, hence NonPremultiplied.
        GraphicsDevice.SetRenderTarget(_frame);
        GraphicsDevice.Clear(Color.Black);
        batch.Begin(blendState: BlendState.NonPremultiplied, samplerState: SamplerState.PointClamp);
        SessionRenderer.Draw(_canvas!, _session, _debug);
        batch.End();

        // 2. Scale it to the window by the largest integer factor that fits, centred.
        GraphicsDevice.SetRenderTarget(null);
        GraphicsDevice.Clear(Color.Black);
        batch.Begin(samplerState: SamplerState.PointClamp);
        batch.Draw(_frame!, LetterboxedDestination(), Color.White);
        batch.End();

        base.Draw(gameTime);
    }

    private Rectangle LetterboxedDestination()
    {
        var viewport = GraphicsDevice.Viewport;
        var scale = Math.Max(1, Math.Min(viewport.Width / _width, viewport.Height / _height));
        var width = _width * scale;
        var height = _height * scale;
        return new Rectangle((viewport.Width - width) / 2, (viewport.Height - height) / 2, width, height);
    }

    protected override void UnloadContent()
    {
        _frame?.Dispose();
        _pixel?.Dispose();
        _spriteBatch?.Dispose();
        base.UnloadContent();
    }
}
