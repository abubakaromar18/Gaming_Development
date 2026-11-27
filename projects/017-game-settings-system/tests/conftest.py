import pytest

from gamesettings.core import SettingsStore, build_schema


@pytest.fixture
def schema():
    return build_schema()


@pytest.fixture
def store(schema):
    return SettingsStore(schema, confirm_timeout=15.0)
