"""Qt Widgets desktop UI. SPDX-License-Identifier: AGPL-3.0-only."""
from __future__ import annotations

import sys
import time
import re
from pathlib import Path

from PySide6.QtCore import Qt, QThread, Signal, QObject, QRunnable, QThreadPool, QTimer, QSettings, QUrl, QStandardPaths, QSize
from PySide6.QtGui import QColor, QDesktopServices, QIcon, QPainter, QPen, QPixmap, QKeySequence, QShortcut
from PySide6.QtWidgets import (QApplication, QMainWindow, QWidget, QFrame, QVBoxLayout, QHBoxLayout,
    QLabel, QPushButton, QPlainTextEdit, QStackedWidget, QScrollArea, QFileDialog, QMessageBox,
    QProgressBar, QMenu, QSystemTrayIcon, QSizePolicy, QScroller)

from . import __version__, REPOSITORY
from .protocol import FileHelperClient, ProtocolError, SessionExpired, Cancelled, Message, safe_filename
from .updates import check_for_update

TEXT = {
    "title": ("文件传输助手", "File Transfer"), "subtitle": ("手机与电脑，轻松互传", "Between your phone and desktop"),
    "chat": ("文件传输", "Conversation"), "files": ("本次会话文件", "Session files"),
    "folder": ("打开下载文件夹", "Open downloads"), "about": ("关于与开源代码", "About & source code"),
    "scan": ("用手机微信扫码", "Scan with WeChat"),
    "scan_hint": ("打开微信「扫一扫」，然后在手机上确认登录。", "Open Scan in WeChat, then confirm on your phone."),
    "session_hint": ("登录信息仅在本次运行中保留，退出后需重新扫码。", "Sign-in lasts for this run. Scan again after restarting."),
    "refresh": ("刷新二维码", "Refresh QR code"), "signout": ("退出登录", "Sign out"),
    "waiting": ("等待扫码", "Waiting for scan"), "confirm": ("已扫码，请在手机上确认", "Scanned · confirm on your phone"),
    "starting": ("正在连接微信…", "Connecting to WeChat…"), "online": ("已连接", "Connected"),
    "reconnecting": ("连接中断，正在重试…", "Connection interrupted · retrying…"),
    "expired": ("二维码或会话已失效，请刷新", "Code or session expired · refresh to sign in"),
    "error": ("连接失败，请检查网络后重试", "Connection failed · check your network and retry"),
    "stopping": ("正在结束会话…", "Ending session…"),
    "placeholder": ("输入文字，或将文件拖到这里…", "Write a message, or drop files here…"),
    "attach": ("添加文件", "Add files"), "image": ("发送图片", "Add image"),
    "send": ("发送", "Send"), "clear": ("移除附件", "Remove attachments"),
    "empty": ("连接后，从这里开始传输", "Your next transfer starts here"),
    "empty_hint": ("发送文字或文件，它们会出现在手机微信的文件传输助手中。", "Send text or files to File Transfer in WeChat on your phone."),
    "files_empty": ("本次会话还没有文件", "No files in this session yet"),
    "hint": ("Ctrl / ⌘ + Enter 发送 · 单个文件上限 25 MiB", "Ctrl / ⌘ + Enter to send · Up to 25 MiB per file"),
    "download": ("另存为…", "Save as…"), "saved": ("已保存", "Saved"),
    "open_folder": ("所在文件夹", "Show folder"), "copy": ("复制", "Copy"),
    "sent": ("已提交微信服务", "Accepted by WeChat"), "received": ("收到", "Received"),
    "you": ("你", "You"), "phone": ("手机微信", "WeChat on phone"),
    "sending": ("正在发送…", "Sending…"), "downloading": ("正在下载…", "Downloading…"),
    "failure": ("操作未完成", "Operation not completed"),
    "pending": ("已添加 {n} 个附件", "{n} attachments ready"),
    "notice": ("第三方开源客户端 · AGPL-3.0", "Independent open-source client · AGPL-3.0"),
    "privacy": ("消息仅保留在本次会话；下载的文件由你选择保存位置。", "Messages stay in this session. You choose where downloads are saved."),
    "quit_title": ("传输仍在进行", "Transfer in progress"),
    "quit_body": ("退出会中断当前操作。确定退出吗？", "Quitting interrupts the current operation. Quit anyway?"),
    "beta": ("预览版", "Preview"),
    "classroom": ("大屏模式", "Large screen"), "exit_classroom": ("退出大屏 · Esc", "Exit full screen · Esc"),
    "updates": ("检查更新", "Check for updates"), "checking": ("正在检查更新…", "Checking for updates…"),
    "current": ("当前已是最新版本", "You are up to date"),
    "new_version": ("发现新版本 {version} · 查看更新", "Update {version} available · View release"),
    "update_failed": ("暂时无法检查更新，点击重试", "Update check failed · click to retry"),
}


def app_icon(size=64) -> QIcon:
    pixmap = QPixmap(size, size)
    pixmap.fill(Qt.GlobalColor.transparent)
    painter = QPainter(pixmap)
    painter.setRenderHint(QPainter.RenderHint.Antialiasing)
    painter.setPen(Qt.PenStyle.NoPen)
    painter.setBrush(QColor("#0067c0"))
    painter.drawRoundedRect(0, 0, size, size, size * .24, size * .24)
    painter.setPen(QPen(QColor("white"), max(2, size * .055), Qt.PenStyle.SolidLine, Qt.PenCapStyle.RoundCap, Qt.PenJoinStyle.RoundJoin))
    for x1, y1, x2, y2 in ((.25, .37, .75, .37), (.62, .24, .75, .37), (.62, .50, .75, .37),
                           (.75, .65, .25, .65), (.38, .52, .25, .65), (.38, .78, .25, .65)):
        painter.drawLine(int(size*x1), int(size*y1), int(size*x2), int(size*y2))
    painter.end()
    return QIcon(pixmap)


def label(text="", name="", wrap=False):
    result = QLabel(text)
    result.setTextFormat(Qt.TextFormat.PlainText)
    result.setWordWrap(wrap)
    if name:
        result.setObjectName(name)
    return result


def size_text(size):
    if size >= 1024 * 1024:
        return f"{size / 1024 / 1024:.1f} MiB"
    if size >= 1024:
        return f"{size / 1024:.1f} KiB"
    return f"{size} B"


class LoginWorker(QThread):
    qr = Signal(bytes)
    state = Signal(str)
    ready = Signal(str)
    message = Signal(object)
    failure = Signal(str)

    def __init__(self, client, parent=None):
        super().__init__(parent)
        self.client = client

    def run(self):
        try:
            self.qr.emit(self.client.create_qr())
            self.state.emit("waiting")
            deadline = time.monotonic() + 180
            scanned = False
            while not self.client.cancel.is_set():
                state = self.client.poll_login(scanned)
                if state == "online":
                    self.ready.emit(self.client.nickname)
                    self.state.emit("online")
                    break
                if state == "confirm":
                    scanned = True
                self.state.emit("confirm" if scanned and state == "waiting" else state)
                if state == "expired" or time.monotonic() > deadline:
                    self.state.emit("expired")
                    return
                self.client.cancel.wait(.5)
            failures = 0
            while not self.client.cancel.is_set() and self.client.logged_in:
                try:
                    for message in self.client.receive():
                        self.message.emit(message)
                    if failures:
                        self.state.emit("online")
                    failures = 0
                    self.client.cancel.wait(.3)
                except SessionExpired:
                    self.state.emit("expired")
                    return
                except Cancelled:
                    return
                except ProtocolError:
                    failures += 1
                    self.state.emit("reconnecting")
                    self.client.cancel.wait(min(30, 2 ** min(failures, 5)))
        except Cancelled:
            pass
        except ProtocolError as exc:
            self.failure.emit(str(exc))
            self.state.emit("error")
        except Exception:
            self.failure.emit("Unexpected response; please retry / 服务响应异常，请重试")
            self.state.emit("error")
        finally:
            # Close only after any active transfer has stopped; the window owns lifetime.
            self.client.logged_in = False


class TaskSignals(QObject):
    success = Signal(object)
    failure = Signal(str)
    progress = Signal(int)
    finished = Signal()


class Task(QRunnable):
    def __init__(self, function):
        super().__init__()
        self.function = function
        self.signals = TaskSignals()

    def run(self):
        try:
            self.signals.success.emit(self.function(self.signals.progress.emit))
        except ProtocolError as exc:
            self.signals.failure.emit(str(exc))
        except OSError:
            self.signals.failure.emit("Cannot read or write this file / 无法读取或保存文件，请检查权限和磁盘空间")
        except Exception:
            self.signals.failure.emit("Unexpected response; operation not retried / 响应异常，未自动重试，请检查手机端结果")
        finally:
            self.signals.finished.emit()


class Composer(QPlainTextEdit):
    files_dropped = Signal(list)
    send_requested = Signal()

    def canInsertFromMimeData(self, source):
        return source.hasUrls() or source.hasText()

    def insertFromMimeData(self, source):
        if source.hasUrls():
            paths = [Path(url.toLocalFile()) for url in source.urls() if url.isLocalFile()]
            self.files_dropped.emit(paths)
        elif source.hasText():
            super().insertFromMimeData(source)

    def keyPressEvent(self, event):
        if event.key() in (Qt.Key.Key_Return, Qt.Key.Key_Enter) and event.modifiers() & Qt.KeyboardModifier.ControlModifier:
            self.send_requested.emit()
            return
        super().keyPressEvent(event)


STYLE = """
QWidget { color: #202b26; font-size: 14px; }
QMainWindow, QWidget#Main { background: #fafbf9; }
QFrame#Rail { background: #eef1ed; border-right: 1px solid #dde3dc; }
QFrame#Header { background: #fafbf9; border-bottom: 1px solid #e0e5de; }
QLabel#Heading { font-size: 23px; font-weight: 600; }
QLabel#Brand { font-size: 18px; font-weight: 600; }
QLabel#Muted { color: #566359; font-size: 12px; }
QLabel#State { color: #176b40; font-size: 13px; }
QLabel#Error { color: #9e392b; font-size: 13px; }
QLabel#QR { background: white; border: 1px solid #dde3dc; border-radius: 12px; }
QPushButton { background: #ffffff; border: 1px solid #cfd8ce; border-radius: 6px; padding: 8px 14px; }
QPushButton:hover { background: #e9f0e7; border-color: #8caa93; }
QPushButton:pressed { background: #dbe8d9; }
QPushButton:focus { border: 2px solid #16834a; padding: 7px 13px; }
QPushButton:disabled { color: #758077; background: #eef0ec; border-color: #dce2d9; }
QPushButton#Primary { background: #16834a; color: white; border-color: #16834a; font-weight: 600; }
QPushButton#Primary:hover { background: #0c6a39; }
QPushButton#Primary:disabled { color: #fafbf9; background: #849d89; border-color: #849d89; }
QPushButton#Nav { text-align: left; background: transparent; border: 1px solid transparent; padding: 12px; }
QPushButton#Nav:checked { background: #dce8da; color: #125c34; font-weight: 600; }
QPushButton#Nav:hover { background: #e3e9e0; }
QPushButton#Quiet { background: transparent; border: 1px solid transparent; color: #46634d; padding: 6px; }
QPushButton#Quiet:hover { background: #e4eee1; }
QPlainTextEdit { background: white; border: 1px solid #cdd7cb; border-radius: 8px; padding: 12px; selection-background-color: #c3e5bf; }
QPlainTextEdit:focus { border-color: #16834a; }
QScrollArea { border: none; background: transparent; }
QWidget#Stream { background: #fafbf9; }
QFrame#Outgoing { background: #dcefd6; border-radius: 10px; }
QFrame#Incoming { background: #edf0e9; border-radius: 10px; }
QFrame#Composer { background: #fafbf9; border-top: 1px solid #e0e5de; }
QProgressBar { border: none; background: #e6ece2; border-radius: 3px; height: 6px; text-align: center; }
QProgressBar::chunk { background: #16834a; border-radius: 3px; }
QScrollBar:vertical { background: transparent; width: 10px; margin: 2px; }
QScrollBar::handle:vertical { background: #bac6b7; min-height: 24px; border-radius: 4px; }
QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical { height: 0px; }
QScrollBar::add-page:vertical, QScrollBar::sub-page:vertical { background: transparent; }
QToolTip { background: #ffffff; color: #202b26; border: 1px solid #cfd8ce; padding: 5px; }
"""

# Fluent-inspired desktop surfaces; this remains a Qt application, not a UWP package.
for _before, _after in {
    "#202b26": "#1b1b1b", "#fafbf9": "#fafafa", "#eef1ed": "#f3f3f3",
    "#dde3dc": "#e5e5e5", "#e0e5de": "#e5e5e5", "#566359": "#5b5b5b",
    "#176b40": "#005a9e", "#16834a": "#0067c0", "#0c6a39": "#005a9e",
    "#e9f0e7": "#f0f0f0", "#8caa93": "#8a8a8a", "#dbe8d9": "#e9e9e9",
    "#758077": "#777777", "#eef0ec": "#f0f0f0", "#dce2d9": "#dedede",
    "#849d89": "#8497a8", "#dce8da": "#e3effb", "#125c34": "#005a9e",
    "#e3e9e0": "#eaeaea", "#46634d": "#333333", "#e4eee1": "#eaeaea",
    "#cdd7cb": "#cecece", "#c3e5bf": "#b6d9f5", "#dcefd6": "#e1effa",
    "#edf0e9": "#eeeeee", "#e6ece2": "#e9e9e9", "#bac6b7": "#b5b5b5",
    "#cfd8ce": "#d1d1d1",
}.items():
    STYLE = STYLE.replace(_before, _after)


class MainWindow(QMainWindow):
    def __init__(self, *, auto_connect=True):
        super().__init__()
        self.settings = QSettings("wxfilehelper-desktop", "FileTransfer")
        self.language = self.settings.value("language", "zh")
        self.client = None
        self.worker = None
        self.active_task = None
        self.task_success_handler = None
        self.messages: list[Message] = []
        self.message_ids = set()
        self.pending: list[tuple[Path, bool]] = []
        self.state = "starting"
        self.nickname = ""
        self.files_only = False
        self.closing = False
        self.restart = False
        self.classroom = False
        self.qr_pixmap = None
        self.update_task = None
        self.update_result = None
        self.pool = QThreadPool(self)
        self.pool.setMaxThreadCount(2)  # An update check must not hold up a transfer.
        self.setWindowIcon(app_icon())
        self.resize(1040, 740)
        self.setMinimumSize(760, 600)
        self.setStyleSheet(STYLE)
        self._build()
        self._translate()
        self._render_messages()
        self._set_state("starting")
        QShortcut(QKeySequence.StandardKey.Open, self, activated=self._choose_files)
        QShortcut(QKeySequence("F11"), self, activated=self._toggle_classroom)
        QShortcut(QKeySequence("Escape"), self, activated=lambda: self._toggle_classroom() if self.classroom else None)
        if QSystemTrayIcon.isSystemTrayAvailable():
            self.tray = QSystemTrayIcon(app_icon(), self)
            self.tray.setToolTip("WX FileHelper Desktop")
            menu = QMenu(self)
            menu.addAction("显示 / Show", self._show)
            menu.addAction("退出 / Quit", self.close)
            self.tray.setContextMenu(menu)
            self.tray.activated.connect(lambda reason: self._show() if reason == QSystemTrayIcon.ActivationReason.DoubleClick else None)
            self.tray.show()
        if auto_connect:
            QTimer.singleShot(100, self.start_login)
            QTimer.singleShot(5000, self._check_update)
            self.update_timer = QTimer(self)
            self.update_timer.timeout.connect(self._check_update)
            self.update_timer.start(6 * 60 * 60 * 1000)

    def tr_key(self, key):
        return TEXT[key][0 if self.language == "zh" else 1]

    def _build(self):
        root = QWidget(objectName="Main")
        self.setCentralWidget(root)
        layout = QHBoxLayout(root)
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(0)
        rail = QFrame(objectName="Rail")
        self.rail = rail
        rail.setFixedWidth(210)
        left = QVBoxLayout(rail)
        left.setContentsMargins(20, 30, 20, 22)
        left.setSpacing(8)
        logo = QLabel()
        logo.setPixmap(app_icon().pixmap(48, 48))
        left.addWidget(logo)
        left.addSpacing(10)
        self.brand = label(name="Brand")
        left.addWidget(self.brand)
        self.preview = label(name="Muted")
        left.addWidget(self.preview)
        left.addSpacing(28)
        self.chat_button = QPushButton(objectName="Nav", checkable=True, checked=True)
        self.files_button = QPushButton(objectName="Nav", checkable=True)
        self.chat_button.clicked.connect(lambda: self._filter(False))
        self.files_button.clicked.connect(lambda: self._filter(True))
        left.addWidget(self.chat_button)
        left.addWidget(self.files_button)
        left.addStretch()
        self.folder_button = QPushButton(objectName="Quiet")
        self.folder_button.clicked.connect(self._downloads)
        self.about_button = QPushButton(objectName="Quiet")
        self.about_button.clicked.connect(self._about)
        left.addWidget(self.folder_button)
        self.update_button = QPushButton(objectName="Quiet")
        self.update_button.clicked.connect(self._check_update)
        left.addWidget(self.update_button)
        left.addWidget(self.about_button)
        left.addSpacing(12)
        left.addWidget(label("AGPL-3.0  ·  v" + __version__, "Muted", True))
        layout.addWidget(rail)
        main = QWidget()
        right = QVBoxLayout(main)
        right.setContentsMargins(0, 0, 0, 0)
        right.setSpacing(0)
        header = QFrame(objectName="Header")
        header_layout = QHBoxLayout(header)
        header_layout.setContentsMargins(26, 21, 24, 19)
        titles = QVBoxLayout()
        self.heading = label(name="Heading")
        self.status = label(name="State", wrap=True)
        titles.addWidget(self.heading)
        titles.addWidget(self.status)
        header_layout.addLayout(titles, 1)
        self.language_button = QPushButton(objectName="Quiet")
        self.language_button.clicked.connect(self._toggle_language)
        self.signout_button = QPushButton(objectName="Quiet")
        self.signout_button.clicked.connect(self._signout)
        self.classroom_button = QPushButton(objectName="Quiet")
        self.classroom_button.clicked.connect(self._toggle_classroom)
        header_layout.addWidget(self.classroom_button)
        header_layout.addWidget(self.language_button)
        header_layout.addWidget(self.signout_button)
        right.addWidget(header)
        self.update_banner = QPushButton(objectName="Quiet")
        self.update_banner.setVisible(False)
        self.update_banner.clicked.connect(self._open_update)
        right.addWidget(self.update_banner)
        self.pages = QStackedWidget()
        login = QWidget()
        login_layout = QVBoxLayout(login)
        login_layout.setContentsMargins(35, 24, 35, 28)
        login_layout.setSpacing(12)
        login_layout.addStretch()
        self.scan_title = label(name="Heading")
        self.scan_title.setAlignment(Qt.AlignmentFlag.AlignCenter)
        login_layout.addWidget(self.scan_title)
        self.scan_hint = label(name="Muted", wrap=True)
        self.scan_hint.setAlignment(Qt.AlignmentFlag.AlignCenter)
        login_layout.addWidget(self.scan_hint)
        login_layout.addSpacing(8)
        self.qr_label = label(name="QR")
        self.qr_label.setFixedSize(228, 228)
        self.qr_label.setAlignment(Qt.AlignmentFlag.AlignCenter)
        login_layout.addWidget(self.qr_label, alignment=Qt.AlignmentFlag.AlignHCenter)
        self.login_status = label(name="State", wrap=True)
        self.login_status.setAlignment(Qt.AlignmentFlag.AlignCenter)
        login_layout.addWidget(self.login_status)
        self.refresh_button = QPushButton(objectName="Primary")
        self.refresh_button.clicked.connect(self.start_login)
        login_layout.addWidget(self.refresh_button, alignment=Qt.AlignmentFlag.AlignHCenter)
        self.login_error = label(name="Error", wrap=True)
        self.login_error.setAlignment(Qt.AlignmentFlag.AlignCenter)
        login_layout.addWidget(self.login_error)
        login_layout.addStretch()
        self.session_hint = label(name="Muted", wrap=True)
        self.session_hint.setAlignment(Qt.AlignmentFlag.AlignCenter)
        login_layout.addWidget(self.session_hint)
        self.pages.addWidget(login)
        chat = QWidget()
        chat_layout = QVBoxLayout(chat)
        chat_layout.setContentsMargins(0, 0, 0, 0)
        chat_layout.setSpacing(0)
        self.scroll = QScrollArea(widgetResizable=True)
        self.stream = QWidget(objectName="Stream")
        self.stream_layout = QVBoxLayout(self.stream)
        self.stream_layout.setContentsMargins(26, 26, 26, 26)
        self.stream_layout.setSpacing(18)
        self.scroll.setWidget(self.stream)
        QScroller.grabGesture(self.scroll.viewport(), QScroller.ScrollerGestureType.TouchGesture)
        chat_layout.addWidget(self.scroll, 1)
        composer_frame = QFrame(objectName="Composer")
        composer_layout = QVBoxLayout(composer_frame)
        composer_layout.setContentsMargins(24, 14, 24, 16)
        toolbar = QHBoxLayout()
        self.attach_button = QPushButton(objectName="Quiet")
        self.attach_button.clicked.connect(self._choose_files)
        self.image_button = QPushButton(objectName="Quiet")
        self.image_button.clicked.connect(self._choose_image)
        toolbar.addWidget(self.attach_button)
        toolbar.addWidget(self.image_button)
        toolbar.addStretch()
        self.clear_button = QPushButton(objectName="Quiet")
        self.clear_button.clicked.connect(self._clear_attachments)
        toolbar.addWidget(self.clear_button)
        composer_layout.addLayout(toolbar)
        self.pending_label = label(name="Muted", wrap=True)
        composer_layout.addWidget(self.pending_label)
        self.composer = Composer()
        self.composer.setFixedHeight(95)
        self.composer.setAccessibleName("Message / 消息")
        self.composer.files_dropped.connect(self._stage_files)
        self.composer.send_requested.connect(self._send)
        composer_layout.addWidget(self.composer)
        self.progress = QProgressBar(textVisible=False)
        self.progress.setFixedHeight(6)
        self.progress.hide()
        composer_layout.addWidget(self.progress)
        bottom = QHBoxLayout()
        self.composer_hint = label(name="Muted", wrap=True)
        bottom.addWidget(self.composer_hint, 1)
        self.send_button = QPushButton(objectName="Primary")
        self.send_button.setMinimumWidth(92)
        self.send_button.clicked.connect(self._send)
        bottom.addWidget(self.send_button)
        composer_layout.addLayout(bottom)
        chat_layout.addWidget(composer_frame)
        self.pages.addWidget(chat)
        right.addWidget(self.pages, 1)
        self.footer = label(name="Muted")
        self.footer.setContentsMargins(26, 8, 24, 10)
        right.addWidget(self.footer)
        layout.addWidget(main, 1)

    def _translate(self):
        for widget, key in ((self.brand, "title"), (self.heading, "title"), (self.chat_button, "chat"),
                (self.files_button, "files"), (self.folder_button, "folder"), (self.about_button, "about"),
                (self.scan_title, "scan"), (self.scan_hint, "scan_hint"), (self.refresh_button, "refresh"),
                (self.signout_button, "signout"), (self.attach_button, "attach"), (self.image_button, "image"),
                (self.clear_button, "clear"), (self.send_button, "send"), (self.composer_hint, "hint"),
                (self.session_hint, "session_hint"), (self.footer, "notice")):
            widget.setText(self.tr_key(key))
        self.setWindowTitle(self.tr_key("title") + " · WX FileHelper Desktop")
        self.preview.setText(self.tr_key("beta") + "  " + __version__)
        self.language_button.setText("English" if self.language == "zh" else "简体中文")
        self.classroom_button.setText(self.tr_key("exit_classroom" if self.classroom else "classroom"))
        self.update_button.setText(self.tr_key("updates"))
        if self.update_result:
            self.update_banner.setText(self.tr_key("new_version").format(version=self.update_result.version))
        self.composer.setPlaceholderText(self.tr_key("placeholder"))
        self._pending_label()
        self._set_state(self.state)

    def _toggle_language(self):
        self.language = "en" if self.language == "zh" else "zh"
        self.settings.setValue("language", self.language)
        self._translate()
        self._render_messages()

    def _set_state(self, state):
        self.state = state
        text = self.tr_key(state)
        self.status.setText(text + (" · " + self.nickname if self.nickname and not self.classroom else ""))
        self.login_status.setText(text)
        self.signout_button.setVisible(state in ("online", "reconnecting") and not self.classroom)
        self.refresh_button.setEnabled(state in ("waiting", "confirm", "expired", "error"))
        enabled = state == "online" and self.active_task is None
        for widget in (self.send_button, self.attach_button, self.image_button, self.composer, self.clear_button):
            widget.setEnabled(enabled)
        if state in ("expired", "error", "stopping"):
            self.qr_pixmap = None
            self.qr_label.clear()
            self.qr_label.setText(self.tr_key(state))
            self.qr_label.setWordWrap(True)
            self.pages.setCurrentIndex(0)

    def start_login(self):
        if self.closing or self.active_task:
            return
        if self.worker and self.worker.isRunning():
            self.restart = True
            self.client.cancel.set()
            self._set_state("stopping")
            return
        if self.client:
            self.client.close()
        self.restart = False
        self.nickname = ""
        self.messages.clear()
        self.message_ids.clear()
        self._clear_attachments()
        self.composer.clear()
        self._render_messages()
        self.pages.setCurrentIndex(0)
        self.qr_label.clear()
        self.qr_label.setText(self.tr_key("starting"))
        self.login_error.clear()
        self._set_state("starting")
        self.client = FileHelperClient()
        self.worker = LoginWorker(self.client, self)
        self.worker.qr.connect(self._qr)
        self.worker.state.connect(self._worker_state)
        self.worker.ready.connect(self._ready)
        self.worker.message.connect(self._add_message)
        self.worker.failure.connect(self.login_error.setText)
        self.worker.finished.connect(self._worker_finished)
        self.worker.start()

    def _qr(self, data):
        if self.closing or self.restart:
            return
        pixmap = QPixmap()
        if not pixmap.loadFromData(data):
            self.login_error.setText("Invalid QR image / 二维码图片无效")
            self.client.cancel.set()
            self._set_state("error")
            return
        self.qr_pixmap = pixmap
        self._fit_qr()

    def _fit_qr(self):
        if self.qr_pixmap:
            size = self.qr_label.width() - 20
            self.qr_label.setPixmap(self.qr_pixmap.scaled(size, size, Qt.AspectRatioMode.KeepAspectRatio, Qt.TransformationMode.FastTransformation))

    def _toggle_classroom(self):
        self.classroom = not self.classroom
        height = self.screen().availableGeometry().height()
        scale = 1.65 if height >= 900 else 1.3
        style = re.sub(r"font-size: (\d+)px", lambda m: f"font-size: {round(int(m[1]) * scale)}px", STYLE) if self.classroom else STYLE
        if self.classroom:
            style += "QPushButton { min-height: 40px; padding: 10px 20px; } QScrollBar:vertical { width: 18px; }"
        self.setStyleSheet(style)
        self.rail.setVisible(not self.classroom)
        self.language_button.setVisible(not self.classroom)
        self.footer.setVisible(not self.classroom)
        self.composer.setFixedHeight(130 if self.classroom else 95)
        qr_size = min(380, max(228, int(height * .35))) if self.classroom else 228
        self.qr_label.setFixedSize(qr_size, qr_size)
        self._fit_qr()
        self._translate()
        self._render_messages()
        self.showFullScreen() if self.classroom else self.showNormal()

    def _check_update(self):
        if self.update_task or self.closing:
            return
        self.update_button.setText(self.tr_key("checking"))
        self.update_task = Task(lambda progress: check_for_update())
        self.update_task.signals.success.connect(self._update_checked)
        self.update_task.signals.failure.connect(self._update_failed)
        self.update_task.signals.finished.connect(self._update_finished)
        self.pool.start(self.update_task)

    def _update_checked(self, result):
        self.update_result = result
        self.update_button.setText(self.tr_key("updates" if result else "current"))
        self.update_banner.setVisible(result is not None)
        if result:
            self.update_banner.setText(self.tr_key("new_version").format(version=result.version))

    def _update_failed(self, error):
        self.update_button.setText(self.tr_key("update_failed"))

    def _update_finished(self):
        self.update_task = None

    def _open_update(self):
        if self.update_result:
            QDesktopServices.openUrl(QUrl(self.update_result.url))

    def _worker_state(self, state):
        if not self.closing and not self.restart:
            self._set_state(state)

    def _ready(self, nickname):
        if self.closing or self.restart:
            return
        self.nickname = nickname
        self.pages.setCurrentIndex(1)
        self.composer.setFocus()

    def _worker_finished(self):
        if self.restart and not self.closing:
            QTimer.singleShot(0, self.start_login)

    def _signout(self):
        if self.active_task:
            return
        self.restart = True
        self.client.cancel.set()
        self.messages.clear()
        self.message_ids.clear()
        self.nickname = ""
        self._set_state("stopping")
        self._render_messages()

    def _filter(self, files):
        self.files_only = files
        self.chat_button.setChecked(not files)
        self.files_button.setChecked(files)
        self._render_messages()

    def _render_messages(self):
        while self.stream_layout.count():
            item = self.stream_layout.takeAt(0)
            if item.widget():
                item.widget().deleteLater()
        messages = [m for m in self.messages if not self.files_only or m.kind in ("file", "image", "video", "voice")]
        if not messages:
            self.stream_layout.addStretch()
            title = label(self.tr_key("files_empty" if self.files_only else "empty"), "Brand", True)
            title.setAlignment(Qt.AlignmentFlag.AlignCenter)
            self.stream_layout.addWidget(title)
            hint = label(self.tr_key("privacy" if self.files_only else "empty_hint"), "Muted", True)
            hint.setAlignment(Qt.AlignmentFlag.AlignCenter)
            self.stream_layout.addWidget(hint)
            self.stream_layout.addStretch()
            return
        for message in messages[-300:]:
            row = QWidget()
            row_layout = QHBoxLayout(row)
            row_layout.setContentsMargins(0, 0, 0, 0)
            bubble = QFrame(objectName="Outgoing" if message.outgoing else "Incoming")
            bubble.setMaximumWidth(900 if self.classroom else 530)
            box = QVBoxLayout(bubble)
            box.setContentsMargins(14, 10, 14, 10)
            box.setSpacing(7)
            who = self.tr_key("you" if message.outgoing else "phone")
            box.addWidget(label(who + "  ·  " + time.strftime("%H:%M", time.localtime(message.timestamp)), "Muted"))
            content = label(message.text, wrap=True)
            content.setTextInteractionFlags(Qt.TextInteractionFlag.TextSelectableByMouse | Qt.TextInteractionFlag.TextSelectableByKeyboard)
            content.setMinimumWidth(100)
            box.addWidget(content)
            if message.kind in ("file", "image", "video", "voice"):
                box.addWidget(label(size_text(message.size) if message.size else message.kind.upper(), "Muted"))
                action = QPushButton(self.tr_key("open_folder" if message.local_path else "download"), objectName="Quiet")
                action.clicked.connect(lambda checked=False, m=message: self._file_action(m))
                box.addWidget(action, alignment=Qt.AlignmentFlag.AlignLeft)
            elif message.kind == "text":
                copy = QPushButton(self.tr_key("copy"), objectName="Quiet")
                copy.clicked.connect(lambda checked=False, text=message.text: QApplication.clipboard().setText(text))
                box.addWidget(copy, alignment=Qt.AlignmentFlag.AlignLeft)
            if message.outgoing:
                box.addWidget(label(self.tr_key("sent"), "Muted"))
                row_layout.addStretch(1)
                row_layout.addWidget(bubble, 5)
            else:
                row_layout.addWidget(bubble, 5)
                row_layout.addStretch(1)
            self.stream_layout.addWidget(row)
        self.stream_layout.addStretch()
        QTimer.singleShot(0, lambda: self.scroll.verticalScrollBar().setValue(self.scroll.verticalScrollBar().maximum()))

    def _add_message(self, message):
        if self.closing or self.restart:
            return
        if message.id in self.message_ids:
            # A sync echo can beat the send response. Merge local acknowledgement
            # so the bubble keeps its desktop direction and local file path.
            if message.outgoing:
                for index, existing in enumerate(self.messages):
                    if existing.id == message.id:
                        self.messages[index] = message
                        self._render_messages()
                        break
            return
        self.message_ids.add(message.id)
        self.messages.append(message)
        # Keep memory bounded. Older messages are never silently written to disk.
        if len(self.messages) > 1000:
            removed = self.messages.pop(0)
            self.message_ids.discard(removed.id)
        self._render_messages()

    def _choose_files(self):
        if self.state != "online" or self.active_task:
            return
        names, _ = QFileDialog.getOpenFileNames(self, self.tr_key("attach"))
        self._stage_files([Path(name) for name in names])

    def _choose_image(self):
        if self.state != "online" or self.active_task:
            return
        names, _ = QFileDialog.getOpenFileNames(self, self.tr_key("image"), "", "Images (*.png *.jpg *.jpeg *.bmp)")
        self._stage_files([Path(name) for name in names], True)

    def _stage_files(self, paths, image=False):
        if self.state != "online" or self.active_task:
            return
        for path in paths:
            if path.is_file() and all(existing != path for existing, _ in self.pending):
                self.pending.append((path, image))
        self._pending_label()

    def _clear_attachments(self):
        self.pending.clear()
        self._pending_label()

    def _pending_label(self):
        self.pending_label.setVisible(bool(self.pending))
        self.clear_button.setVisible(bool(self.pending))
        self.pending_label.setText(self.tr_key("pending").format(n=len(self.pending)) + " · " + "、".join(p.name for p, _ in self.pending))

    def _run_task(self, function, success, label_key):
        if self.active_task or self.closing:
            return
        task = Task(function)
        self.active_task = task
        self.task_success_handler = success
        task.signals.success.connect(self._task_success)
        task.signals.failure.connect(self._task_failure)
        task.signals.progress.connect(self.progress.setValue)
        task.signals.finished.connect(self._task_finished)
        self.progress.setValue(0)
        self.progress.show()
        self.composer_hint.setText(self.tr_key(label_key))
        self.signout_button.setEnabled(False)
        self._set_state(self.state)
        self.pool.start(task)

    def _task_finished(self):
        self.active_task = None
        self.task_success_handler = None
        self.progress.hide()
        self.composer_hint.setText(self.tr_key("hint"))
        self.signout_button.setEnabled(True)
        self._set_state(self.state)

    def _task_success(self, result):
        # Route callbacks through a QObject method so all widget updates stay on the GUI thread.
        if not self.closing and self.task_success_handler:
            self.task_success_handler(result)

    def _task_failure(self, error):
        if not self.closing:
            QMessageBox.warning(self, self.tr_key("failure"), error)

    def _send(self):
        if self.state != "online" or self.active_task:
            return
        text = self.composer.toPlainText()
        pending = list(self.pending)
        if not text.strip() and not pending:
            return
        # Send sequentially, acknowledge each item independently. Never auto-retry a send.
        # Failed and unsent attachments remain staged; acknowledged ones are removed.
        task = Task(lambda progress: None)
        def work(progress):
            if text.strip():
                task.signals.success.emit(("text", self.client.send_text(text), None))
            for path, as_image in pending:
                message = self.client.send_file(path, progress, image=as_image)
                task.signals.success.emit(("file", message, path))
            return None
        def acknowledged(result):
            if result is None:
                return
            kind, message, path = result
            self._add_message(message)
            if kind == "text":
                self.composer.clear()
            else:
                self.pending = [(p, i) for p, i in self.pending if p != path]
                self._pending_label()
        task.function = work
        self.active_task = task
        self.task_success_handler = acknowledged
        task.signals.success.connect(self._task_success)
        task.signals.failure.connect(self._task_failure)
        task.signals.progress.connect(self.progress.setValue)
        task.signals.finished.connect(self._task_finished)
        self.progress.setValue(0)
        self.progress.show()
        self.composer_hint.setText(self.tr_key("sending"))
        self.signout_button.setEnabled(False)
        self._set_state(self.state)
        self.pool.start(task)

    def _file_action(self, message):
        if message.local_path:
            QDesktopServices.openUrl(QUrl.fromLocalFile(str(Path(message.local_path).parent)))
            return
        if self.state != "online" or self.active_task:
            return
        base = self.settings.value("downloads", QStandardPaths.writableLocation(QStandardPaths.StandardLocation.DownloadLocation))
        target, _ = QFileDialog.getSaveFileName(self, self.tr_key("download"), str(Path(base) / safe_filename(message.text)))
        if not target:
            return
        self.settings.setValue("downloads", str(Path(target).parent))
        def done(path):
            message.local_path = str(path)
            self._render_messages()
        self._run_task(lambda progress: self.client.download(message, Path(target), progress), done, "downloading")

    def _downloads(self):
        folder = self.settings.value("downloads", QStandardPaths.writableLocation(QStandardPaths.StandardLocation.DownloadLocation))
        QDesktopServices.openUrl(QUrl.fromLocalFile(folder))

    def _about(self):
        box = QMessageBox(self)
        box.setWindowTitle(self.tr_key("about"))
        box.setTextFormat(Qt.TextFormat.RichText)
        box.setText(f"<b>WX FileHelper Desktop {__version__}</b><p>原生 Qt 客户端 / Native Qt client</p>"
                    "<p>第三方项目，与腾讯无隶属关系。<br>Independent project; not affiliated with Tencent.</p>"
                    "<p>AGPL-3.0-only · Qt/PySide6: LGPL-3.0</p>"
                    f'<p><a href="{REPOSITORY}">源码与许可证 / Source & licenses</a></p>'
                    f'<p><a href="{REPOSITORY}/releases">下载与更新 / Downloads & updates</a></p>')
        box.setInformativeText("预览版使用非公开稳定承诺的服务接口，可能失效。请备份重要文件。\n"
                               "Preview using undocumented interfaces, which may change. Back up important files.\n"
                               "免责声明 / Disclaimer: " + REPOSITORY + "/blob/main/DISCLAIMER.md")
        box.exec()

    def _show(self):
        self.showNormal()
        self.raise_()
        self.activateWindow()

    def closeEvent(self, event):
        if self.active_task and not self.closing:
            answer = QMessageBox.question(self, self.tr_key("quit_title"), self.tr_key("quit_body"))
            if answer != QMessageBox.StandardButton.Yes:
                event.ignore()
                return
        self.closing = True
        self.restart = False
        if self.client:
            self.client.cancel.set()
        if hasattr(self, "tray"):
            self.tray.hide()
        event.accept()
        self.hide()
        self.shutdown_timer = QTimer(self)
        self.shutdown_timer.timeout.connect(self._finish_shutdown)
        self.shutdown_timer.start(100)

    def _finish_shutdown(self):
        if (not self.worker or not self.worker.isRunning()) and self.pool.activeThreadCount() == 0:
            if self.client:
                self.client.close()
            self.shutdown_timer.stop()
            QApplication.instance().quit()
