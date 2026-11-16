"""Drive the real curses UI through a pseudo-terminal (POSIX only)."""
import os
import select
import sys
import time
from pathlib import Path

import pytest

pty = pytest.importorskip("pty")
pytest.importorskip("curses")

SRC = Path(__file__).resolve().parents[1] / "src"


def _read_until(fd: int, needle: bytes, timeout: float) -> bytes:
    out = b""
    deadline = time.monotonic() + timeout
    while needle not in out and time.monotonic() < deadline:
        ready, _, _ = select.select([fd], [], [], 0.05)
        if ready:
            try:
                out += os.read(fd, 65536)
            except OSError:
                break
    return out


def test_autoplay_solves_level_in_real_terminal():
    pid, fd = pty.fork()
    if pid == 0:  # child: become the game
        os.environ.update(TERM="xterm", PYTHONPATH=str(SRC), LINES="30", COLUMNS="100")
        os.execv(sys.executable, [sys.executable, "-m", "sokoban", "--level", "2"])
    try:
        assert b"Side Step" in _read_until(fd, b"Side Step", 10)
        os.write(fd, b"p")
        assert b"Solved in" in _read_until(fd, b"Solved in", 10)
        os.write(fd, b"q")
        _, status = os.waitpid(pid, 0)
        assert os.waitstatus_to_exitcode(status) == 0
    finally:
        try:
            os.kill(pid, 9)
        except ProcessLookupError:
            pass
        os.close(fd)
