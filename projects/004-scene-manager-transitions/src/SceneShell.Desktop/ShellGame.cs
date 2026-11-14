using Microsoft.Xna.Framework;
using Microsoft.Xna.Framework.Graphics;
using Microsoft.Xna.Framework.Input;
using SceneShell.Core.Arcade;
using SceneShell.Core.Randomness;

namespace SceneShell.Desktop;

/// <summary>
/// The MonoGame host. It owns the window and GPU resources and nothing else: each frame it
/// polls the keyboard, hands (dt, input) to the core <see cref="ArcadeShell"/>, and draws
/// the shell into a low-resolution render target that is scaled up with point sampling
/// and letterboxed to the window.
/// </summary>
public sealed class ShellGame : Game
{
    private const int WindowScale = 4;

    private readonly ArcadeConfig _config;
    private readonly ArcadeShell _shell;
    private readonly KeyboardInput _keyboard = new();

    private SpriteBatch? _spriteBatch;
    private Texture2D? _pixel;
    private RenderTarget2D? _frame;
    private SpriteBatchCanvas? _canvas;

    public ShellGame(ArcadeConfig config, ulong seed)
    {
        _config = config ?? throw new ArgumentNullException(nameof(config));
        _shell = new ArcadeShell(config, new Pcg32(seed));

        // The manager registers itself with the Game; its preferred sizes are applied
        // when MonoGame creates the graphics device during Initialize.
        _ = new GraphicsDeviceManager(this)
        {
            PreferredBackBufferWidth = config.Width * WindowScale,
            PreferredBackBufferHeight = config.Height * WindowScale,
            SynchronizeWithVerticalRetrace = true,
        };

        // The core takes a variable dt, so let MonoGame run one update per rendered frame.
        IsFixedTimeStep = false;
        IsMouseVisible = true;
        Window.AllowUserResizing = true;
        Window.Title = "Scene Shell";
    }

    protected override void LoadContent()
    {
        _spriteBatch = new SpriteBatch(GraphicsDevice);
        _pixel = new Texture2D(GraphicsDevice, 1, 1);
        _pixel.SetData(new[] { Color.White });
        _frame = new RenderTarget2D(GraphicsDevice, _config.Width, _config.Height);
        _canvas = new SpriteBatchCanvas(_spriteBatch, _pixel, _config.Width, _config.Height);
    }

    protected override void Update(GameTime gameTime)
    {
        if (!IsActive)
        {
            // Ignore keys while unfocused and drop held state, so nothing sticks on return.
            _keyboard.Reset();
            base.Update(gameTime);
            return;
        }

        var input = _keyboard.Poll(Keyboard.GetState());
        _shell.Update((float)gameTime.ElapsedGameTime.TotalSeconds, input);
        if (_shell.HasQuit)
        {
            Exit();
        }
        base.Update(gameTime);
    }

    protected override void Draw(GameTime gameTime)
    {
        var batch = _spriteBatch!;

        // 1. Draw the logical 320×180 frame. Colors use straight alpha, hence NonPremultiplied.
        GraphicsDevice.SetRenderTarget(_frame);
        GraphicsDevice.Clear(Color.Black);
        batch.Begin(blendState: BlendState.NonPremultiplied, samplerState: SamplerState.PointClamp);
        _shell.Draw(_canvas!);
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
        var scale = Math.Max(1, Math.Min(viewport.Width / _config.Width, viewport.Height / _config.Height));
        var width = _config.Width * scale;
        var height = _config.Height * scale;
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
