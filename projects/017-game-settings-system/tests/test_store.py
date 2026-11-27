import pytest

from gamesettings.core import RuleViolationError, SettingsStore, StateError, ValidationError


def test_starts_from_defaults_plus_overrides(schema):
    store = SettingsStore(schema, {"graphics.vsync": False})
    assert store.get("graphics.vsync") is False
    assert store.get("audio.master_volume") == 80


def test_constructor_rejects_conflicting_values(schema):
    with pytest.raises(RuleViolationError):
        SettingsStore(schema, {"controls.bind_jump": "E"})


def test_values_view_is_read_only(store):
    with pytest.raises(TypeError):
        store.values["graphics.vsync"] = False


def test_staging_previews_without_touching_live(store):
    store.stage("graphics.vsync", False)
    assert store.preview("graphics.vsync") is False
    assert store.get("graphics.vsync") is True
    assert dict(store.pending) == {"graphics.vsync": False}


def test_staging_back_to_live_value_clears_pending(store):
    store.stage("audio.music_volume", 50)
    store.stage("audio.music_volume", 70)
    assert not store.pending


def test_adjust_uses_staged_value(store):
    store.adjust("audio.music_volume", 1)
    store.adjust("audio.music_volume", 1)
    assert store.preview("audio.music_volume") == 80


def test_adjust_rejects_typed_only_fields(store):
    with pytest.raises(ValidationError):
        store.adjust("controls.bind_jump", 1)


def test_stage_many_is_all_or_nothing(store):
    with pytest.raises(ValidationError):
        store.stage_many({"graphics.vsync": False, "audio.music_volume": 900})
    assert not store.pending


def test_apply_commits_and_notifies(store):
    seen = []
    store.subscribe(seen.append)
    store.stage("graphics.vsync", False)
    store.stage("audio.sfx_volume", 50)
    result = store.apply()
    assert store.get("graphics.vsync") is False
    assert {c.key for c in result.changes} == {"graphics.vsync", "audio.sfx_volume"}
    assert seen == [result.changes]
    assert not result.awaiting_confirmation and not store.pending


def test_apply_with_nothing_staged_notifies_nobody(store):
    seen = []
    store.subscribe(seen.append)
    assert store.apply().changes == ()
    assert seen == []


def test_apply_reports_restart_required(store):
    store.stage("graphics.texture_quality", "low")
    assert store.apply().restart_required == ("graphics.texture_quality",)


def test_rule_violation_keeps_edits_staged_for_fixing(store):
    store.stage_text("controls.bind_jump", "e")
    with pytest.raises(RuleViolationError) as info:
        store.apply()
    assert "controls.bind_interact" in info.value.violations[0].keys
    assert store.get("controls.bind_jump") == "SPACE"
    store.stage_text("controls.bind_interact", "f")
    store.apply()
    assert store.get("controls.bind_jump") == "E"


def test_swapping_two_bindings_in_one_apply_is_fine(store):
    store.stage_many({"controls.bind_jump": "E", "controls.bind_interact": "SPACE"})
    store.apply()
    assert store.get("controls.bind_interact") == "SPACE"


def test_unsubscribe(store):
    seen = []
    unsubscribe = store.subscribe(seen.append)
    unsubscribe()
    unsubscribe()  # idempotent
    store.stage("graphics.vsync", False)
    store.apply()
    assert seen == []


def test_stage_defaults(schema):
    store = SettingsStore(schema, {"audio.music_volume": 10, "graphics.vsync": False})
    store.stage_defaults(["audio.music_volume"])
    assert dict(store.pending) == {"audio.music_volume": 70}
    store.discard()
    store.stage_defaults()
    assert set(store.pending) == {"audio.music_volume", "graphics.vsync"}


class TestConfirmation:
    def apply_resolution(self, store):
        store.stage("graphics.resolution", "2560x1440")
        return store.apply()

    def test_risky_change_goes_live_but_not_durable(self, store):
        result = self.apply_resolution(store)
        assert result.awaiting_confirmation
        assert store.get("graphics.resolution") == "2560x1440"
        assert store.durable_values["graphics.resolution"] == "1920x1080"

    def test_confirm_makes_it_durable(self, store):
        self.apply_resolution(store)
        store.update(10.0)
        store.confirm()
        assert store.durable_values["graphics.resolution"] == "2560x1440"
        assert store.update(100.0) is False

    def test_timeout_reverts_and_notifies(self, store):
        seen = []
        store.subscribe(seen.append)
        self.apply_resolution(store)
        assert store.update(14.9) is False
        assert store.seconds_remaining == pytest.approx(0.1)
        assert store.update(0.2) is True
        assert store.get("graphics.resolution") == "1920x1080"
        assert seen[-1][0].new == "1920x1080"
        assert not store.awaiting_confirmation

    def test_revert_restores_everything_applied_together(self, store):
        store.stage_many({"graphics.window_mode": "fullscreen", "audio.music_volume": 20})
        store.apply()
        store.revert()
        assert store.get("audio.music_volume") == 70
        assert store.get("graphics.window_mode") == "borderless"

    def test_no_edits_while_awaiting(self, store):
        self.apply_resolution(store)
        with pytest.raises(StateError):
            store.stage("graphics.vsync", False)
        with pytest.raises(StateError):
            store.apply()

    def test_confirm_and_revert_need_a_pending_change(self, store):
        with pytest.raises(StateError):
            store.confirm()
        with pytest.raises(StateError):
            store.revert()

    def test_safe_changes_skip_confirmation(self, store):
        store.stage("graphics.vsync", False)
        assert not store.apply().awaiting_confirmation

    def test_opt_out_of_confirmation(self, store):
        store.stage("graphics.resolution", "1280x720")
        assert not store.apply(require_confirmation=False).awaiting_confirmation

    def test_negative_dt_rejected(self, store):
        with pytest.raises(ValueError):
            store.update(-1)

    def test_timeout_must_be_positive(self, schema):
        with pytest.raises(ValueError):
            SettingsStore(schema, confirm_timeout=0)
