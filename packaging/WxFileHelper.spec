# SPDX-License-Identifier: AGPL-3.0-only
import os
from pathlib import Path
from runpy import run_path

root = Path(SPECPATH).parent
version = run_path(str(root / "wxfilehelper" / "__init__.py"))["__version__"]
onefile = os.environ.get("WX_ONEFILE") == "1"
name = f"WXFileHelper-{version}-Windows-x64-Portable" if onefile else "WXFileHelper"
data = [(str(root / "LICENSE"), "."), (str(root / "DISCLAIMER.md"), "."),
        (str(root / "THIRD_PARTY_NOTICES.md"), "."), (str(root / "licenses"), "licenses")]
a = Analysis([str(root / "run.py")], pathex=[str(root)], binaries=[], datas=data,
             hiddenimports=[], hookspath=[], hooksconfig={}, runtime_hooks=[],
             excludes=["PySide6.QtWebEngineCore", "PySide6.QtWebEngineWidgets", "PySide6.QtWebEngineQuick",
                       "PySide6.QtQml", "PySide6.QtQuick", "PySide6.QtPdf", "PySide6.Qt3DCore",
                       "tkinter", "pytest", "PIL"], noarchive=False)
pyz = PYZ(a.pure)
options = dict(name=name, debug=False, bootloader_ignore_signals=False, strip=False,
               upx=False, console=False, disable_windowed_traceback=False,
               icon=str(root / "build" / "app.ico"))
if onefile:
    exe = EXE(pyz, a.scripts, a.binaries, a.datas, [], **options)
else:
    exe = EXE(pyz, a.scripts, [], exclude_binaries=True, **options)
    coll = COLLECT(exe, a.binaries, a.datas, strip=False, upx=False, name=name)
