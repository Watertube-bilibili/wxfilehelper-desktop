# v0.2.0-beta.3 · Windows 启动修复 / Windows startup fix

## 中文

- 修复旧 EXE 受其他软件的 Qt 平台设置影响而提示 “Reinstalling the application may fix this problem” 的启动故障。安装版和便携版均使用包内的 Windows 平台插件及同一版本的 Visual C++ 运行库。
- 安装版覆盖升级时清理早期包遗留的两个 ICU 文件。遇到系统版本过低、安装包损坏等情况时，安装程序会给出中英文提示。启动时检测到 Qt 组件加载错误，弹窗会列出组件与错误信息，并尝试将诊断写入 `%LOCALAPPDATA%\WXFileHelper\startup-error.txt`。
- 适用系统为 Windows 10 1809（内部版本 17763）及以上、Windows 11，x64。退出旧版后运行新的 `Setup.exe` 覆盖安装，或直接运行新的 `Portable.exe`。可用 `SHA256SUMS.txt` 校验。预览版未签名。

## English

- Fixed a reproducible launch failure in which a Qt platform setting inherited from another program triggered “Reinstalling the application may fix this problem.” Both packages use their own Windows platform plugin and a matching set of Visual C++ libraries.
- Setup removes two obsolete ICU files when upgrading an early installation. Setup provides bilingual messages for an unsupported Windows build or damaged installer. If the app detects a Qt component load failure, the dialog shows the component and error details. It also attempts to save a report at `%LOCALAPPDATA%\WXFileHelper\startup-error.txt`.
- Requires Windows 10 version 1809 (build 17763) or later, or Windows 11, x64. Quit the old app and run the new `Setup.exe` over your installation, or run the new `Portable.exe`. Verify the download with `SHA256SUMS.txt`. Preview binaries are unsigned.

[中英双语免责声明 / Bilingual disclaimer](https://github.com/Watertube-bilibili/wxfilehelper-desktop/blob/main/DISCLAIMER.md)
