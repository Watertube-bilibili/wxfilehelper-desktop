import os
import sys
import traceback
from pathlib import Path


if __name__ == "__main__":
    smoke = "--smoke-test" in sys.argv
    report = os.environ.get("WXFILEHELPER_SMOKE_REPORT") if smoke else None
    try:
        if sys.platform == "win32" and getattr(sys, "frozen", False):
            from wxfilehelper.windows_startup import WindowsStartupError, prepare_qt, show_startup_error
            try:
                prepare_qt(Path(sys._MEIPASS))
                from wxfilehelper.app import main
            except (WindowsStartupError, ImportError) as error:
                if smoke:
                    raise
                show_startup_error(error)
                raise SystemExit(1)
        else:
            from wxfilehelper.app import main
        code = main()
    except Exception:
        if not report:
            raise
        Path(report).write_text(traceback.format_exc(), encoding="utf-8")
        code = 1
    else:
        if report:
            Path(report).write_text("OK\n" if code == 0 else f"Exit {code}\n", encoding="utf-8")
    raise SystemExit(code)
