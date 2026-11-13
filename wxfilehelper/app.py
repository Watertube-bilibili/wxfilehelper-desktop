"""Application entry point. SPDX-License-Identifier: AGPL-3.0-only."""
import sys
from PySide6.QtWidgets import QApplication
from PySide6.QtGui import QFont
from .ui import MainWindow


def main():
    app = QApplication(sys.argv)
    app.setApplicationName("WX FileHelper Desktop")
    app.setOrganizationName("wxfilehelper-desktop")
    app.setQuitOnLastWindowClosed(False)
    font = QFont("Segoe UI" if sys.platform == "win32" else ".AppleSystemUIFont", 10)
    app.setFont(font)
    window = MainWindow()
    window.show()
    return app.exec()


if __name__ == "__main__":
    raise SystemExit(main())
