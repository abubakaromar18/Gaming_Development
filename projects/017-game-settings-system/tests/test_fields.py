import math

import pytest

from gamesettings.core import BoolField, ChoiceField, FloatField, IntField, KeyField, ResolutionField, ValidationError


def bound(field, key="t.x"):
    field.bind(key)
    return field


class TestBool:
    def test_accepts_bools_and_words(self):
        f = bound(BoolField(False, "B"))
        assert f.coerce(True) is True
        assert f.parse("On") is True
        assert f.parse(" no ") is False

    @pytest.mark.parametrize("bad", [1, 0, "maybe", None, 1.0])
    def test_rejects_non_bools(self, bad):
        with pytest.raises(ValidationError):
            bound(BoolField(False, "B")).coerce(bad)

    def test_step_toggles(self):
        f = bound(BoolField(False, "B"))
        assert f.step(False, 1) is True and f.step(True, -1) is False


class TestNumbers:
    def test_int_range_and_units(self):
        f = bound(IntField(50, "Vol", minimum=0, maximum=100, step=5, unit="%"))
        assert f.parse("40%") == 40
        assert f.coerce(60.0) == 60
        with pytest.raises(ValidationError, match="outside"):
            f.coerce(101)

    @pytest.mark.parametrize("bad", [True, "loud", 2.5, None, math.nan, math.inf])
    def test_int_rejects(self, bad):
        with pytest.raises(ValidationError):
            bound(IntField(50, "Vol", minimum=0, maximum=100, step=5)).coerce(bad)

    def test_clamp_pulls_into_range(self):
        f = bound(IntField(90, "FOV", minimum=60, maximum=120, step=5))
        assert f.coerce(130, clamp=True) == 120
        assert f.coerce(10, clamp=True) == 60

    def test_step_snaps_to_grid_and_clamps(self):
        f = bound(IntField(50, "Vol", minimum=0, maximum=100, step=5))
        assert f.step(42, 1) == 45
        assert f.step(100, 1) == 100
        assert f.step(0, -1) == 0

    def test_float_steps_do_not_accumulate_error(self):
        f = bound(FloatField(1.0, "Scale", minimum=0.5, maximum=2.0, step=0.05))
        value = 0.5
        for _ in range(10):
            value = f.step(value, 1)
        assert value == 1.0

    def test_default_is_validated_on_bind(self):
        with pytest.raises(ValidationError):
            bound(IntField(500, "Vol", minimum=0, maximum=100, step=5))

    def test_bad_ranges_are_programmer_errors(self):
        with pytest.raises(ValueError):
            IntField(5, "X", minimum=10, maximum=0, step=1)
        with pytest.raises(ValueError):
            FloatField(1.0, "X", minimum=0, maximum=2, step=0)


class TestChoice:
    def test_case_insensitive_canonical(self):
        f = bound(ChoiceField("high", "Q", choices=("low", "high")))
        assert f.coerce("HIGH") == "high"

    def test_step_wraps(self):
        f = bound(ChoiceField("a", "Q", choices=("a", "b", "c")))
        assert f.step("c", 1) == "a"
        assert f.step("a", -1) == "c"

    def test_rejects_unknown(self):
        with pytest.raises(ValidationError, match="expected one of"):
            bound(ChoiceField("a", "Q", choices=("a", "b"))).coerce("z")


class TestResolution:
    def test_normalizes(self):
        f = bound(ResolutionField("1920x1080", "R"))
        assert f.parse(" 2560 X 1440 ") == "2560x1440"
        assert f.coerce("1024×768") == "1024x768"

    @pytest.mark.parametrize("bad", ["big", "100x100", "99999x99999", 1920, "1920x"])
    def test_rejects(self, bad):
        with pytest.raises(ValidationError):
            bound(ResolutionField("1920x1080", "R")).coerce(bad)

    def test_step_walks_common_modes_from_odd_sizes(self):
        f = bound(ResolutionField("1920x1080", "R"))
        assert f.step("1920x1080", 1) == "2560x1440"
        assert f.step("1920x1080", -1) == "1600x900"
        assert f.step("1700x1000", 1) == "1920x1080"
        assert f.step("3840x2160", 1) == "3840x2160"


class TestKey:
    def test_aliases_and_case(self):
        f = bound(KeyField("SPACE", "Jump"))
        assert f.coerce("esc") == "ESCAPE"
        assert f.coerce("q") == "Q"
        assert not f.steppable

    def test_unknown_key(self):
        with pytest.raises(ValidationError, match="unknown key"):
            bound(KeyField("SPACE", "Jump")).coerce("HYPERDRIVE")
        with pytest.raises(TypeError):
            bound(KeyField("SPACE", "Jump")).step("SPACE", 1)
