from breakout.core.actions import Action
from breakout.core.bindings import Key, PadAxis, PadButton, PadHat
from breakout.core.capture import CaptureConfig, CaptureStatus, RebindCapture
from breakout.core.resolver import NO_INPUT
from helpers import keys, pad, raw

CANCEL = [Key("escape"), PadButton(1)]


def capture(start=NO_INPUT, **cfg):
    return RebindCapture(Action.LAUNCH, start, CANCEL, CaptureConfig(**cfg))


def test_captures_a_fresh_key():
    c = capture()
    assert c.feed(NO_INPUT, 0.1) is CaptureStatus.WAITING
    assert c.feed(keys("z"), 0.1) is CaptureStatus.CAPTURED
    assert c.captured == Key("z")


def test_input_held_when_the_prompt_opened_is_ignored_until_released():
    # The player pressed Enter to open the prompt and is still holding it.
    c = capture(start=keys("return"))
    assert c.feed(keys("return"), 0.1) is CaptureStatus.WAITING
    assert c.feed(NO_INPUT, 0.1) is CaptureStatus.WAITING
    # Pressing it again after release is a deliberate choice: capture it.
    assert c.feed(keys("return"), 0.1) is CaptureStatus.CAPTURED
    assert c.captured == Key("return")


def test_fresh_input_is_captured_even_while_old_input_is_still_held():
    c = capture(start=pad(buttons=[0]))
    assert c.feed(pad(buttons=[0, 3]), 0.1) is CaptureStatus.CAPTURED
    assert c.captured == PadButton(3)


def test_cancel_source_cancels():
    c = capture()
    assert c.feed(keys("escape"), 0.1) is CaptureStatus.CANCELLED
    assert c.captured is None


def test_stick_needs_a_deliberate_push():
    c = capture(axis_threshold=0.6)
    assert c.feed(pad(axes={0: -0.4}), 0.1) is CaptureStatus.WAITING
    assert c.feed(pad(axes={0: -0.8}), 0.1) is CaptureStatus.CAPTURED
    assert c.captured == PadAxis(0, -1)


def test_hats_are_capturable():
    c = capture()
    c.feed(pad(hats=[(0, "right")]), 0.1)
    assert c.captured == PadHat(0, "right")


def test_times_out_without_input():
    c = capture(timeout_seconds=1.0)
    for _ in range(9):
        assert c.feed(NO_INPUT, 0.1) is CaptureStatus.WAITING
    assert c.feed(NO_INPUT, 0.2) is CaptureStatus.TIMED_OUT
    assert c.seconds_left == 0.0


def test_status_is_sticky_once_finished():
    c = capture()
    c.feed(keys("z"), 0.1)
    assert c.feed(keys("x"), 0.1) is CaptureStatus.CAPTURED
    assert c.captured == Key("z")


def test_simultaneous_presses_resolve_deterministically():
    c = capture()
    c.feed(raw(keys=["k", "b"], buttons=[2]), 0.1)
    assert c.captured == Key("b")
