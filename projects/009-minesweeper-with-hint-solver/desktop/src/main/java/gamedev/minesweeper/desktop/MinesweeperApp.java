package gamedev.minesweeper.desktop;

import com.badlogic.gdx.ApplicationAdapter;
import com.badlogic.gdx.Gdx;
import com.badlogic.gdx.Input;
import com.badlogic.gdx.InputAdapter;
import com.badlogic.gdx.graphics.OrthographicCamera;
import com.badlogic.gdx.graphics.g2d.BitmapFont;
import com.badlogic.gdx.graphics.g2d.SpriteBatch;
import com.badlogic.gdx.graphics.glutils.ShapeRenderer;
import com.badlogic.gdx.math.Vector3;
import com.badlogic.gdx.utils.ScreenUtils;
import gamedev.minesweeper.core.BoardConfig;
import gamedev.minesweeper.core.CellView;
import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.Game;
import gamedev.minesweeper.core.GameStatus;
import gamedev.minesweeper.solver.Hint;
import gamedev.minesweeper.solver.HintEngine;
import gamedev.minesweeper.solver.ProbabilityMap;

import java.util.SplittableRandom;

/**
 * The libGDX application: owns the window resources, turns input into game actions,
 * feeds frame time into {@link Game#tick(double)} and asks the renderer to draw.
 * All rules live in the core module; this class only wires them to the platform.
 */
final class MinesweeperApp extends ApplicationAdapter {

    private static final String CONTROLS =
            "Left: open / chord   Right: flag   H: hint   P: odds   R: retry seed   N: new   1-3: level";
    /** Clamp for frame time so a stalled window (dragging, breakpoints) does not jump the clock. */
    private static final float MAX_FRAME_SECONDS = 0.25f;

    private final HintEngine engine = new HintEngine();
    private final SplittableRandom seeds;

    private BoardConfig config;
    private long seed;
    private Game game;
    private Layout layout;
    private Hint hint;
    private boolean showOdds;
    private ProbabilityMap odds;

    private OrthographicCamera camera;
    private ShapeRenderer shapes;
    private SpriteBatch batch;
    private BitmapFont font;
    private BoardRenderer renderer;

    MinesweeperApp(LaunchOptions options) {
        this.config = options.config();
        this.seeds = new SplittableRandom(options.seed().orElse(System.nanoTime()));
        this.seed = options.seed().orElseGet(seeds::nextLong);
    }

    @Override
    public void create() {
        camera = new OrthographicCamera();
        shapes = new ShapeRenderer();
        batch = new SpriteBatch();
        font = new BitmapFont(true); // libGDX's bundled default font, flipped for the y-down camera
        renderer = new BoardRenderer(shapes, batch, font);
        startGame();
        Gdx.input.setInputProcessor(new Controls());
    }

    @Override
    public void render() {
        game.tick(Math.min(Gdx.graphics.getDeltaTime(), MAX_FRAME_SECONDS));
        ScreenUtils.clear(0, 0, 0, 1);
        camera.update();
        shapes.setProjectionMatrix(camera.combined);
        batch.setProjectionMatrix(camera.combined);
        renderer.draw(layout, game, hint, showOdds ? odds : null, message());
    }

    @Override
    public void resize(int width, int height) {
        camera.setToOrtho(true, layout.windowWidth(), layout.windowHeight());
    }

    @Override
    public void dispose() {
        shapes.dispose();
        batch.dispose();
        font.dispose();
    }

    // ---------------------------------------------------------------- game flow

    private void startGame() {
        game = new Game(config, new SplittableRandom(seed));
        Layout next = new Layout(config);
        boolean resized = layout == null
                || next.windowWidth() != layout.windowWidth() || next.windowHeight() != layout.windowHeight();
        layout = next;
        if (resized) {
            Gdx.graphics.setWindowedMode(layout.windowWidth(), layout.windowHeight());
            camera.setToOrtho(true, layout.windowWidth(), layout.windowHeight());
        }
        Gdx.graphics.setTitle("Minesweeper with Hint Solver  |  seed " + seed);
        boardChanged();
    }

    /** Any change invalidates the shown hint and (if visible) recomputes the odds overlay. */
    private void boardChanged() {
        hint = null;
        odds = showOdds ? engine.analyze(game).probabilities().orElse(null) : null;
    }

    private String message() {
        if (game.status() == GameStatus.WON) {
            return String.format("Cleared in %.1f s. Press N for a new board.", game.elapsedSeconds());
        }
        if (game.status() == GameStatus.LOST) {
            return "Boom. Press R to retry this seed or N for a new board.";
        }
        if (hint != null) {
            return hint.explanation();
        }
        if (showOdds && odds != null) {
            return String.format("%,d mine layouts fit what you can see. Greener cells are safer.", odds.layouts());
        }
        return CONTROLS;
    }

    private void click(Coord cell, int button) {
        boolean changed = switch (button) {
            case Input.Buttons.LEFT -> game.view(cell).kind() == CellView.Kind.OPEN
                    ? game.chord(cell).changedBoard()
                    : game.reveal(cell).changedBoard();
            case Input.Buttons.RIGHT -> game.toggleFlag(cell);
            case Input.Buttons.MIDDLE -> game.chord(cell).changedBoard();
            default -> false;
        };
        if (changed) {
            boardChanged();
        }
    }

    private final class Controls extends InputAdapter {
        private final Vector3 point = new Vector3();

        @Override
        public boolean touchDown(int screenX, int screenY, int pointer, int button) {
            camera.unproject(point.set(screenX, screenY, 0));
            if (layout.onFace(point.x, point.y)) {
                seed = seeds.nextLong();
                startGame();
                return true;
            }
            Coord cell = layout.cellAt(point.x, point.y);
            if (cell != null) {
                click(cell, button);
            }
            return true;
        }

        @Override
        public boolean keyDown(int keycode) {
            switch (keycode) {
                case Input.Keys.H -> hint = engine.hint(game).orElse(null);
                case Input.Keys.P -> {
                    showOdds = !showOdds;
                    boardChanged();
                }
                case Input.Keys.R -> startGame();
                case Input.Keys.N -> {
                    seed = seeds.nextLong();
                    startGame();
                }
                case Input.Keys.NUM_1 -> changeLevel(BoardConfig.BEGINNER);
                case Input.Keys.NUM_2 -> changeLevel(BoardConfig.INTERMEDIATE);
                case Input.Keys.NUM_3 -> changeLevel(BoardConfig.EXPERT);
                case Input.Keys.ESCAPE -> Gdx.app.exit();
                default -> {
                    return false;
                }
            }
            return true;
        }

        private void changeLevel(BoardConfig level) {
            config = level;
            seed = seeds.nextLong();
            startGame();
        }
    }
}
