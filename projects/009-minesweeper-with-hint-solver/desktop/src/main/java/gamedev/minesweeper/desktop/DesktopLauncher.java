package gamedev.minesweeper.desktop;

import com.badlogic.gdx.backends.lwjgl3.Lwjgl3Application;
import com.badlogic.gdx.backends.lwjgl3.Lwjgl3ApplicationConfiguration;

/** Entry point: parses arguments and opens a window sized for the chosen board. */
public final class DesktopLauncher {

    private DesktopLauncher() {
    }

    public static void main(String[] args) {
        LaunchOptions options;
        try {
            options = LaunchOptions.parse(args);
        } catch (IllegalArgumentException e) {
            System.err.println(e.getMessage());
            System.err.println(LaunchOptions.USAGE);
            System.exit(2);
            return;
        }
        Layout layout = new Layout(options.config());
        Lwjgl3ApplicationConfiguration window = new Lwjgl3ApplicationConfiguration();
        window.setTitle("Minesweeper with Hint Solver");
        window.setWindowedMode(layout.windowWidth(), layout.windowHeight());
        window.setResizable(false);
        window.useVsync(true);
        window.setForegroundFPS(60);
        new Lwjgl3Application(new MinesweeperApp(options), window);
    }
}
