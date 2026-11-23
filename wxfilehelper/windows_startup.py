"""Initialize the frozen Windows Qt runtime. SPDX-License-Identifier: AGPL-3.0-only."""
import ctypes
import os
import sys
from pathlib import Path

from . import __version__


class WindowsStartupError(RuntimeError):
    pass


def prepare_qt(bundle: Path) -> None:
    qt = bundle / "PySide6"
    plugins = qt / "plugins"
    platforms = plugins / "platforms"
    # Other Windows applications can export their own QPA backend or plugins.
    os.environ["QT_PLUGIN_PATH"] = str(plugins)
    os.environ["QT_QPA_PLATFORM_PLUGIN_PATH"] = str(platforms)
    os.environ["QT_QPA_PLATFORM"] = "windows"

    # A missing dependency here otherwise aborts in QApplication before Python
    # can present the name of the failing component or the Windows error code.
    load = ctypes.WinDLL("kernel32", use_last_error=True).LoadLibraryExW
    load.argtypes = (ctypes.c_wchar_p, ctypes.c_void_p, ctypes.c_uint32)
    load.restype = ctypes.c_void_p
    # Search the DLL's own directory, our registered bundle dirs, and System32.
    search_dirs = 0x00000100 | 0x00000400 | 0x00000800
    component = qt
    try:
        with os.add_dll_directory(str(bundle)), os.add_dll_directory(str(qt)):
            for component in (qt / "Qt6Core.dll", qt / "Qt6Gui.dll", platforms / "qwindows.dll"):
                if not component.is_file():
                    raise ctypes.WinError(2)  # ERROR_FILE_NOT_FOUND
                if not load(str(component), None, search_dirs):
                    raise ctypes.WinError(ctypes.get_last_error())
    except OSError as error:
        detail = f"WinError {error.winerror}: {error.strerror}"
        raise WindowsStartupError(f"{component.name}\n{detail}") from error


def verify_loaded_runtime(bundle: Path) -> None:
    """In the packaged smoke test, ensure the GUI did not borrow host libraries."""
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    module = kernel.GetModuleHandleW
    module.argtypes = (ctypes.c_wchar_p,)
    module.restype = ctypes.c_void_p
    filename = kernel.GetModuleFileNameW
    filename.argtypes = (ctypes.c_void_p, ctypes.c_wchar_p, ctypes.c_uint32)
    filename.restype = ctypes.c_uint32
    bundle = bundle.resolve()
    for name in (
        "qwindows.dll", "Qt6Core.dll", "Qt6Gui.dll", "Qt6Widgets.dll",
        "MSVCP140.dll", "MSVCP140_1.dll", "MSVCP140_2.dll", "VCRUNTIME140.dll", "VCRUNTIME140_1.dll",
    ):
        handle = module(name)
        path = ctypes.create_unicode_buffer(32768)
        if not handle or not filename(handle, path, len(path)):
            raise RuntimeError(f"Could not find the loaded module {name}")
        if not Path(path.value).resolve().is_relative_to(bundle):
            raise RuntimeError(f"{name} loaded from outside the bundle: {path.value}")


def show_startup_error(error: Exception) -> None:
    version = sys.getwindowsversion()
    message = (
        "无法启动 Windows 桌面界面。请在反馈时附上以下信息。\n"
        "The Windows interface could not start. Include these details when reporting the issue.\n\n"
        f"WX FileHelper {__version__}  |  Windows {version.major}.{version.minor}.{version.build}\n{error}"
    )
    local = os.environ.get("LOCALAPPDATA")
    if local:
        log = Path(local) / "WXFileHelper" / "startup-error.txt"
        try:
            log.parent.mkdir(parents=True, exist_ok=True)
            log.write_text(message + "\n", encoding="utf-8")
        except OSError:
            pass
        else:
            message += f"\n\n日志 / Log\n{log}"
    ctypes.windll.user32.MessageBoxW(None, message, "WX FileHelper", 0x10)
