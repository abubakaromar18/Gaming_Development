import pytest

from camera2d.core import FEATURES, CameraConfig


@pytest.mark.parametrize(
    "field", ["smooth_time", "deadzone_width", "lookahead_max", "trauma_decay", "shake_frequency"]
)
def test_rejects_negative_tuning(field):
    with pytest.raises(ValueError, match=field):
        CameraConfig(**{field: -1.0})


def test_rejects_bad_zoom_range_and_power():
    with pytest.raises(ValueError):
        CameraConfig(min_zoom=2.0, max_zoom=1.0)
    with pytest.raises(ValueError):
        CameraConfig(min_zoom=0.0)
    with pytest.raises(ValueError):
        CameraConfig(trauma_power=0.0)


def test_with_changes_validates():
    with pytest.raises(ValueError):
        CameraConfig().with_changes(smooth_time=-0.1)
    assert CameraConfig().with_changes(smooth_time=0.5).smooth_time == 0.5


def test_toggle_each_feature():
    base = CameraConfig()
    for field in FEATURES:
        flipped = base.toggled(field)
        assert getattr(flipped, field) is not getattr(base, field)
        assert flipped.toggled(field) == base


def test_toggle_unknown_feature():
    with pytest.raises(KeyError):
        CameraConfig().toggled("smooth_time")
