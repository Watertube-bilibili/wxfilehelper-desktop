import os
os.environ.setdefault("QT_QPA_PLATFORM", "offscreen")

import pytest
from PySide6.QtWidgets import QApplication
from PySide6.QtCore import Qt
from wxfilehelper.ui import MainWindow
from wxfilehelper.protocol import Message


@pytest.fixture(scope="module")
def app():
    return QApplication.instance() or QApplication([])


def test_initial_window_has_no_browser_and_cannot_send(app):
    window = MainWindow(auto_connect=False)
    assert window.pages.currentIndex() == 0
    assert not window.send_button.isEnabled()
    assert window.minimumWidth() == 760
    assert window.heading.textFormat() == Qt.TextFormat.PlainText
    window.deleteLater()


def test_bilingual_ui_and_deduplication(app):
    window = MainWindow(auto_connect=False)
    window.language = "en"
    window._translate()
    assert window.send_button.text() == "Send"
    message = Message("42", "text", "<img src=x>literal text", True)
    window._add_message(message)
    window._add_message(message)
    assert len(window.messages) == 1
    window.language = "zh"
    window._translate()
    assert window.send_button.text() == "发送"
    window.deleteLater()


def test_drop_stages_without_sending(app, tmp_path):
    window = MainWindow(auto_connect=False)
    file = tmp_path / "test.txt"
    file.write_text("test")
    window._set_state("online")
    window._stage_files([file])
    assert window.pending == [(file, False)]
    assert not window.messages
    assert window.active_task is None
    window._clear_attachments()
    assert not window.pending
    window.deleteLater()


def test_classroom_has_visible_exit_and_hides_nickname(app):
    window = MainWindow(auto_connect=False)
    window.nickname = "Private account name"
    window._set_state("online")
    window._toggle_classroom()
    assert window.classroom and window.isFullScreen()
    assert not window.rail.isVisible()
    assert "Private account name" not in window.status.text()
    assert "Esc" in window.classroom_button.text()
    window._toggle_classroom()
    assert not window.classroom
    assert window.qr_label.width() == 228
    window.hide()
    window.deleteLater()


def test_local_ack_merges_sync_echo(app):
    window = MainWindow(auto_connect=False)
    window._add_message(Message("42", "file", "test.txt", False))
    window._add_message(Message("42", "file", "test.txt", True, local_path="test.txt"))
    assert len(window.messages) == 1
    assert window.messages[0].outgoing and window.messages[0].local_path == "test.txt"
    window.deleteLater()
