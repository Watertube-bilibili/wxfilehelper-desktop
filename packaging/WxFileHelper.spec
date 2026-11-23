# SPDX-License-Identifier: AGPL-3.0-only
import os
from importlib.metadata import distribution
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

# The bootloader loads Python's VC runtime before Qt starts. Put the same
# redistributables at the bundle root and at any paths collected by the hooks.
qt_dir = Path(distribution("PySide6-Essentials").locate_file("PySide6"))
runtime_dlls = {
    name: str(qt_dir / name)
    for name in ("msvcp140.dll", "msvcp140_1.dll", "msvcp140_2.dll",
                 "vcruntime140.dll", "vcruntime140_1.dll")
}
for source in runtime_dlls.values():
    if not Path(source).is_file():
        raise RuntimeError(f"Missing Visual C++ runtime from the PySide6 wheel: {source}")
a.binaries = [
    (dest, runtime_dlls.get(Path(dest).name.lower(), source), kind)
    for dest, source, kind in a.binaries if dest.lower() not in runtime_dlls
]
a.binaries.extend((name, source, "BINARY") for name, source in runtime_dlls.items())

collected = {Path(dest).as_posix().lower() for dest, _, _ in a.binaries}
if "pyside6/plugins/platforms/qwindows.dll" not in collected:
    raise RuntimeError("PyInstaller did not collect Qt's Windows platform plugin")
if any(Path(dest).name == "icuuc.dll" for dest in collected):
    raise RuntimeError("Qt needs Windows system ICU. Build with scripts/build_windows.ps1 and a clean PATH.")

pyz = PYZ(a.pure)
options = dict(name=name, debug=False, bootloader_ignore_signals=False, strip=False,
               upx=False, console=False, disable_windowed_traceback=False,
               icon=str(root / "build" / "app.ico"))
if onefile:
    exe = EXE(pyz, a.scripts, a.binaries, a.datas, [], **options)
else:
    exe = EXE(pyz, a.scripts, [], exclude_binaries=True, **options)
    coll = COLLECT(exe, a.binaries, a.datas, strip=False, upx=False, name=name)
