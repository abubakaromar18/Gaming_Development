package gamedev.minesweeper.desktop;

import com.badlogic.gdx.Gdx;
import com.badlogic.gdx.graphics.Color;
import com.badlogic.gdx.graphics.GL20;
import com.badlogic.gdx.graphics.g2d.BitmapFont;
import com.badlogic.gdx.graphics.g2d.GlyphLayout;
import com.badlogic.gdx.graphics.g2d.SpriteBatch;
import com.badlogic.gdx.graphics.glutils.ShapeRenderer;
import com.badlogic.gdx.utils.Align;
import gamedev.minesweeper.core.CellView;
import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.Game;
import gamedev.minesweeper.core.GameStatus;
import gamedev.minesweeper.core.Grid;
import gamedev.minesweeper.solver.Hint;
import gamedev.minesweeper.solver.ProbabilityMap;

/**
 * Draws the whole window from game state. Everything is procedural: bevelled tiles,
 * flags, mines, the face and the seven-segment counters are shapes; text uses
 * libGDX's built-in default font. The renderer only reads state and never mutates it.
 */
final class BoardRenderer {

    private static final Color FACE_BG = new Color(0.75f, 0.75f, 0.75f, 1);
    private static final Color LIGHT = new Color(1f, 1f, 1f, 1);
    private static final Color SHADOW = new Color(0.5f, 0.5f, 0.5f, 1);
    private static final Color GRID_LINE = new Color(0.6f, 0.6f, 0.6f, 1);
    private static final Color LED_ON = new Color(1f, 0.1f, 0.1f, 1);
    private static final Color LED_OFF = new Color(0.3f, 0.02f, 0.02f, 1);
    private static final Color SAFE_HINT = new Color(0.1f, 0.75f, 0.2f, 1);
    private static final Color MINE_HINT = new Color(0.9f, 0.1f, 0.1f, 1);
    private static final Color GUESS_HINT = new Color(1f, 0.7f, 0f, 1);
    private static final Color EVIDENCE = new Color(0.2f, 0.45f, 1f, 1);
    /** Classic number colours, index = number. */
    private static final Color[] NUMBER_COLORS = {
        Color.CLEAR,
        new Color(0f, 0f, 1f, 1), new Color(0f, 0.5f, 0f, 1), new Color(1f, 0f, 0f, 1),
        new Color(0f, 0f, 0.5f, 1), new Color(0.5f, 0f, 0f, 1), new Color(0f, 0.5f, 0.5f, 1),
        new Color(0f, 0f, 0f, 1), new Color(0.5f, 0.5f, 0.5f, 1),
    };
    /** Seven-segment masks for 0-9 and '-', bits a..g = 0..6 (a top, then clockwise, g middle). */
    private static final int[] SEGMENTS = {0x3F, 0x06, 0x5B, 0x4F, 0x66, 0x6D, 0x7D, 0x07, 0x7F, 0x6F};
    private static final int MINUS = 0x40;

    private final ShapeRenderer shapes;
    private final SpriteBatch batch;
    private final BitmapFont font;
    private final GlyphLayout glyphs = new GlyphLayout();

    BoardRenderer(ShapeRenderer shapes, SpriteBatch batch, BitmapFont font) {
        this.shapes = shapes;
        this.batch = batch;
        this.font = font;
    }

    /**
     * @param hint the hint to highlight, or null
     * @param odds probabilities to tint hidden cells with, or null
     * @param message footer text
     */
    void draw(Layout layout, Game game, Hint hint, ProbabilityMap odds, String message) {
        shapes.begin(ShapeRenderer.ShapeType.Filled);
        shapes.setColor(FACE_BG);
        shapes.rect(0, 0, layout.windowWidth(), layout.windowHeight());
        drawHeader(layout, game);
        drawCells(layout, game);
        shapes.end();

        if (odds != null) {
            drawOdds(layout, game, odds);
        }
        if (hint != null) {
            drawHint(layout, hint);
        }

        batch.begin();
        drawNumbers(layout, game);
        font.setColor(Color.BLACK);
        font.draw(batch, message, Layout.MARGIN, layout.footerY() + 6,
                layout.windowWidth() - 2 * Layout.MARGIN, Align.left, true);
        batch.end();
    }

    // ---------------------------------------------------------------- header

    private void drawHeader(Layout layout, Game game) {
        sunken(Layout.MARGIN, Layout.MARGIN, layout.windowWidth() - 2 * Layout.MARGIN, Layout.HEADER, 2);
        counter(layout.mineCounterX(), layout.counterY(), game.remainingMines());
        counter(layout.timerX(), layout.counterY(), (int) Math.min(999, game.elapsedSeconds()));
        face(layout.faceX(), layout.faceY(), game.status());
    }

    private void counter(int x, int y, int value) {
        int clamped = Math.max(-99, Math.min(999, value));
        int w = 3 * (Layout.DIGIT_W + 3) + 3;
        shapes.setColor(Color.BLACK);
        shapes.rect(x - 3, y - 2, w, Layout.DIGIT_H + 4);
        String text = String.format("%03d", Math.abs(clamped));
        if (clamped < 0) {
            text = "-" + String.format("%02d", -clamped);
        }
        for (int i = 0; i < 3; i++) {
            char ch = text.charAt(i);
            int mask = ch == '-' ? MINUS : SEGMENTS[ch - '0'];
            digit(x + i * (Layout.DIGIT_W + 3), y, mask);
        }
    }

    /** One seven-segment digit as seven rectangles; unlit segments glow faintly. */
    private void digit(float x, float y, int mask) {
        float w = Layout.DIGIT_W;
        float h = Layout.DIGIT_H;
        float t = 3;
        float half = h / 2;
        float[][] seg = {
            {x + t, y, w - 2 * t, t},                   // a top
            {x + w - t, y + t, t, half - 1.5f * t},     // b upper right
            {x + w - t, y + half + t / 2, t, half - 1.5f * t}, // c lower right
            {x + t, y + h - t, w - 2 * t, t},           // d bottom
            {x, y + half + t / 2, t, half - 1.5f * t},  // e lower left
            {x, y + t, t, half - 1.5f * t},             // f upper left
            {x + t, y + half - t / 2, w - 2 * t, t},    // g middle
        };
        for (int s = 0; s < 7; s++) {
            shapes.setColor((mask & (1 << s)) != 0 ? LED_ON : LED_OFF);
            shapes.rect(seg[s][0], seg[s][1], seg[s][2], seg[s][3]);
        }
    }

    private void face(float x, float y, GameStatus status) {
        raised(x, y, Layout.FACE, Layout.FACE, 2);
        float cx = x + Layout.FACE / 2f;
        float cy = y + Layout.FACE / 2f;
        float r = Layout.FACE / 2f - 5;
        shapes.setColor(Color.BLACK);
        shapes.circle(cx, cy, r + 1, 24);
        shapes.setColor(Color.YELLOW);
        shapes.circle(cx, cy, r, 24);
        shapes.setColor(Color.BLACK);
        switch (status) {
            case WON -> {
                shapes.rect(cx - r + 2, cy - 4, r - 2, 4);
                shapes.rect(cx + 1, cy - 4, r - 3, 4);
                shapes.rectLine(cx - r + 1, cy - 3, cx + r - 1, cy - 3, 1);
            }
            case LOST -> {
                cross(cx - 4, cy - 3, 2);
                cross(cx + 4, cy - 3, 2);
            }
            default -> {
                shapes.circle(cx - 4, cy - 3, 1.5f, 8);
                shapes.circle(cx + 4, cy - 3, 1.5f, 8);
            }
        }
        // Mouth: a smile curves down (y-down), a frown curves up.
        float bend = status == GameStatus.LOST ? -3 : 3;
        float px = cx - 6;
        float py = cy + 5;
        for (int i = 1; i <= 8; i++) {
            float u = i / 8f;
            float nx = cx - 6 + 12 * u;
            float ny = cy + 5 + bend * (float) Math.sin(Math.PI * u);
            shapes.rectLine(px, py, nx, ny, 1.5f);
            px = nx;
            py = ny;
        }
    }

    private void cross(float cx, float cy, float s) {
        shapes.rectLine(cx - s, cy - s, cx + s, cy + s, 1.2f);
        shapes.rectLine(cx - s, cy + s, cx + s, cy - s, 1.2f);
    }

    // ---------------------------------------------------------------- cells

    private void drawCells(Layout layout, Game game) {
        Grid grid = game.grid();
        for (int y = 0; y < grid.height(); y++) {
            for (int x = 0; x < grid.width(); x++) {
                float px = layout.cellX(x);
                float py = layout.cellY(y);
                CellView view = game.view(new Coord(x, y));
                switch (view.kind()) {
                    case HIDDEN -> raised(px, py, Layout.CELL, Layout.CELL, 3);
                    case FLAGGED -> {
                        raised(px, py, Layout.CELL, Layout.CELL, 3);
                        flag(px, py);
                    }
                    case OPEN -> flat(px, py, FACE_BG);
                    case MINE -> {
                        flat(px, py, FACE_BG);
                        mine(px, py);
                    }
                    case EXPLODED -> {
                        flat(px, py, Color.RED);
                        mine(px, py);
                    }
                    case WRONG_FLAG -> {
                        flat(px, py, FACE_BG);
                        mine(px, py);
                        shapes.setColor(Color.RED);
                        shapes.rectLine(px + 3, py + 3, px + Layout.CELL - 3, py + Layout.CELL - 3, 2);
                        shapes.rectLine(px + 3, py + Layout.CELL - 3, px + Layout.CELL - 3, py + 3, 2);
                    }
                }
            }
        }
    }

    private void drawNumbers(Layout layout, Game game) {
        Grid grid = game.grid();
        for (int y = 0; y < grid.height(); y++) {
            for (int x = 0; x < grid.width(); x++) {
                CellView view = game.view(new Coord(x, y));
                if (view.kind() == CellView.Kind.OPEN && view.adjacent() > 0) {
                    font.setColor(NUMBER_COLORS[view.adjacent()]);
                    glyphs.setText(font, Integer.toString(view.adjacent()));
                    font.draw(batch, glyphs,
                            layout.cellX(x) + (Layout.CELL - glyphs.width) / 2,
                            layout.cellY(y) + (Layout.CELL - glyphs.height) / 2);
                }
            }
        }
    }

    private void flat(float x, float y, Color fill) {
        shapes.setColor(GRID_LINE);
        shapes.rect(x, y, Layout.CELL, Layout.CELL);
        shapes.setColor(fill);
        shapes.rect(x + 1, y + 1, Layout.CELL - 1, Layout.CELL - 1);
    }

    /** A raised tile: light top-left edge, dark bottom-right edge. */
    private void raised(float x, float y, float w, float h, float bevel) {
        shapes.setColor(SHADOW);
        shapes.rect(x, y, w, h);
        shapes.setColor(LIGHT);
        shapes.triangle(x, y, x + w, y, x, y + h);
        shapes.setColor(FACE_BG);
        shapes.rect(x + bevel, y + bevel, w - 2 * bevel, h - 2 * bevel);
    }

    /** A sunken panel: the inverse bevel. */
    private void sunken(float x, float y, float w, float h, float bevel) {
        shapes.setColor(LIGHT);
        shapes.rect(x, y, w, h);
        shapes.setColor(SHADOW);
        shapes.triangle(x, y, x + w, y, x, y + h);
        shapes.setColor(FACE_BG);
        shapes.rect(x + bevel, y + bevel, w - 2 * bevel, h - 2 * bevel);
    }

    private void flag(float x, float y) {
        float c = Layout.CELL;
        shapes.setColor(Color.BLACK);
        shapes.rect(x + c * 0.25f, y + c * 0.72f, c * 0.5f, c * 0.1f);
        shapes.rect(x + c * 0.48f, y + c * 0.2f, c * 0.08f, c * 0.55f);
        shapes.setColor(Color.RED);
        shapes.triangle(x + c * 0.56f, y + c * 0.18f, x + c * 0.56f, y + c * 0.5f, x + c * 0.22f, y + c * 0.34f);
    }

    private void mine(float x, float y) {
        float cx = x + Layout.CELL / 2f;
        float cy = y + Layout.CELL / 2f;
        float r = Layout.CELL * 0.26f;
        shapes.setColor(Color.BLACK);
        shapes.rectLine(cx - r * 1.5f, cy, cx + r * 1.5f, cy, 2);
        shapes.rectLine(cx, cy - r * 1.5f, cx, cy + r * 1.5f, 2);
        shapes.rectLine(cx - r * 1.1f, cy - r * 1.1f, cx + r * 1.1f, cy + r * 1.1f, 2);
        shapes.rectLine(cx - r * 1.1f, cy + r * 1.1f, cx + r * 1.1f, cy - r * 1.1f, 2);
        shapes.circle(cx, cy, r, 16);
        shapes.setColor(Color.WHITE);
        shapes.circle(cx - r * 0.35f, cy - r * 0.35f, r * 0.25f, 8);
    }

    // ---------------------------------------------------------------- overlays

    /** Tints each hidden cell from green (safe) to red (mine) by its exact mine probability. */
    private void drawOdds(Layout layout, Game game, ProbabilityMap odds) {
        Grid grid = game.grid();
        Gdx.gl.glEnable(GL20.GL_BLEND);
        Gdx.gl.glBlendFunc(GL20.GL_SRC_ALPHA, GL20.GL_ONE_MINUS_SRC_ALPHA);
        shapes.begin(ShapeRenderer.ShapeType.Filled);
        for (int i = 0; i < grid.size(); i++) {
            double p = odds.mineProbability(i);
            if (Double.isNaN(p) || game.isRevealed(i)) {
                continue;
            }
            Coord c = grid.coord(i);
            shapes.setColor((float) p, (float) (1 - p), 0.1f, 0.45f);
            shapes.rect(layout.cellX(c.x()) + 3, layout.cellY(c.y()) + 3, Layout.CELL - 6, Layout.CELL - 6);
        }
        shapes.end();
        Gdx.gl.glDisable(GL20.GL_BLEND);
    }

    private void drawHint(Layout layout, Hint hint) {
        shapes.begin(ShapeRenderer.ShapeType.Filled);
        switch (hint) {
            case Hint.FirstClick f -> { }
            case Hint.Deduction d -> {
                for (Coord e : d.evidence()) {
                    outline(layout, e, EVIDENCE);
                }
                outline(layout, d.cell(), d.mine() ? MINE_HINT : SAFE_HINT);
            }
            case Hint.WrongFlag w -> {
                for (Coord e : w.proof().evidence()) {
                    outline(layout, e, EVIDENCE);
                }
                outline(layout, w.cell(), GUESS_HINT);
            }
            case Hint.Guess g -> outline(layout, g.cell(), GUESS_HINT);
        }
        shapes.end();
    }

    private void outline(Layout layout, Coord c, Color color) {
        float x = layout.cellX(c.x());
        float y = layout.cellY(c.y());
        float s = Layout.CELL;
        float t = 3;
        shapes.setColor(color);
        shapes.rect(x, y, s, t);
        shapes.rect(x, y + s - t, s, t);
        shapes.rect(x, y, t, s);
        shapes.rect(x + s - t, y, t, s);
    }
}
