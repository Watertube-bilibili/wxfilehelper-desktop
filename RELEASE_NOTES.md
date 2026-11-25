# 1.2.0 · 原生架构预览 / Native architecture preview

## 中文

Windows 安装版和单文件版共用一个 Go/Win32 可执行程序，无需单独安装 Python、Qt 或 WebView2。最低目标是 Windows 10 1507（内部版本 10240）x64。安装程序使用独立的安装目录。

支持双账号扫码、文字和文件收发、文件队列、系统打开与保存对话框、教室大屏和更新检查。程序不会自动安装更新。macOS 版保留独立仓库。

已验证当前 Windows 环境中的原生启动、安装和卸载，以及传输核心从微信服务获取两张独立登录二维码。1507 客户端上的启动、扫码确认和真实文件收发仍待实测。Windows EXE 为未签名预览包。

## English

The Windows installer and single-file package use the same Go/Win32 executable and require no separate Python, Qt, or WebView2 installation. The compatibility target is x64 Windows 10 version 1507 (build 10240). Setup uses a dedicated installation directory.

Includes independent account tabs, QR authorization, text and file transfers, an outgoing queue, native open/save dialogs, classroom mode, and update checks that link to GitHub. macOS remains in its own repository.

Native startup, setup, and uninstall were verified in the current Windows test environment. Two independent login QR images were received from WeChat using the new transport. Startup, QR confirmation, and live file transfers on Windows 10 1507 still require end-to-end testing. Windows preview EXEs are unsigned.

[免责声明 / Disclaimer](https://github.com/Watertube-bilibili/wxfilehelper-desktop/blob/main/DISCLAIMER.md) · [macOS](https://github.com/Watertube-bilibili/wxfilehelper-macos)
