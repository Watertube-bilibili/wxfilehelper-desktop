"""Regression coverage for frozen Windows Qt startup."""
import importlib.util
import os
import sys
from pathlib import Path

import pytest

from wxfilehelper.windows_startup import WindowsStartupError, prepare_qt

pytestmark = pytest.mark.skipif(sys.platform != "win32", reason="Windows DLL loader")


@pytest.fixture(autouse=True)
def foreign_qt_settings(monkeypatch):
    for key in ("QT_QPA_PLATFORM", "QT_PLUGIN_PATH", "QT_QPA_PLATFORM_PLUGIN_PATH"):
        monkeypatch.setenv(key, "foreign-qt")


def test_bundled_windows_platform_takes_precedence():
    qt = Path(importlib.util.find_spec("PySide6").origin).parent
    prepare_qt(qt.parent)
    assert os.environ["QT_QPA_PLATFORM"] == "windows"
    assert Path(os.environ["QT_PLUGIN_PATH"]) == qt / "plugins"
    assert Path(os.environ["QT_QPA_PLATFORM_PLUGIN_PATH"]) == qt / "plugins" / "platforms"


def test_missing_qt_library_reports_component_and_windows_code(tmp_path):
    (tmp_path / "PySide6").mkdir()
    with pytest.raises(WindowsStartupError, match=r"Qt6Core\.dll\nWinError 2:"):
        prepare_qt(tmp_path)
