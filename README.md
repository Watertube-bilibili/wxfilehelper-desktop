# 微信文件传输助手桌面版 · WX FileHelper Desktop

[中文](#中文) · [English](#english) · [Windows 下载](https://github.com/Watertube-bilibili/wxfilehelper-desktop/releases) · [macOS 仓库](https://github.com/Watertube-bilibili/wxfilehelper-macos) · [免责声明 / Disclaimer](DISCLAIMER.md)

## 中文

1.2 的传输核心使用 Go。Windows 界面采用 Win32，macOS 界面采用 AppKit。软件直接连接微信文件传输服务，通过手机微信扫码授权。本项目独立维护，与腾讯和微信无隶属关系。

### Windows 下载与使用

| 文件 | 用途 |
| --- | --- |
| `WXFileHelper-1.2.0-Windows-x64-Portable.exe` | 单文件，双击运行 |
| `WXFileHelper-1.2.0-Windows-x64-Setup.exe` | 当前用户安装，提供开始菜单和卸载入口 |
| `SHA256SUMS.txt` | 核对两个 EXE 的 SHA-256 |

- 兼容目标为 **Windows 10 1507（内部版本 10240）及以上的 x64 系统**。两个包都无需另装 Python、Qt 或 WebView2。
- 用手机微信扫一扫，确认授权，再在当前标签发送文字或文件。点击 `+` 可添加第二个账号。两个标签各自保留消息和待发送文件。
- 选择、拖入或粘贴文件，核对发送列表后点击「发送」。每批最多 100 个文件，每项 1 字节到 25 MiB。发送失败时未完成的文件仍可重试。
- 选中收到的文件后「另存为」。大屏模式放大主要操作并隐藏昵称。更新检查显示可用版本，点击后可到 GitHub 获取安装包。
- 1.2 安装版使用独立目录和开始菜单项。登录会话驻留内存，首次启动需要重新扫码。

### 从源码构建

需要 x64 Go 1.26 和 Inno Setup 6。项目源码位于 `native/`。首次构建会下载固定版本的 Go 资源编译工具。

```powershell
cd native
go test ./internal/...
cd ..
.\native\packaging\windows\build.ps1 -Verify
```

生成的 EXE 位于 `native/dist/windows/`。若工具位于自定义目录，可传入 `-Go` 或 `-Iscc`。macOS 独立发布，两种架构的构建步骤在 [macOS 仓库](https://github.com/Watertube-bilibili/wxfilehelper-macos)。旧 0.2 的 Python/Qt 源码见 [历史标签](https://github.com/Watertube-bilibili/wxfilehelper-desktop/tree/v0.2.0-beta.3)。

### 隐私与许可

登录票据、消息和发送队列仅在进程内存中保留。收到的文件由用户选择保存位置。微信服务处理扫码和文件传输，GitHub 接收更新检查请求。每个账号须自行扫码授权。微信服务接口、账户条件和网络状态可能影响可用性。[双语免责声明](DISCLAIMER.md)

采用 **AGPL-3.0-only**。 [许可证](LICENSE) · [第三方声明](THIRD_PARTY_NOTICES.md)

## English

Version 1.2 uses a shared Go transfer core with Win32 on Windows and AppKit on macOS. It connects directly to WeChat File Transfer, with sign-in approved by scanning a QR code in WeChat. This independently maintained project is unaffiliated with Tencent and WeChat.

### Download and use on Windows

| Asset | Use |
| --- | --- |
| `WXFileHelper-1.2.0-Windows-x64-Portable.exe` | Single executable. Double-click to launch. |
| `WXFileHelper-1.2.0-Windows-x64-Setup.exe` | Per-user installation with a Start menu and uninstaller. |
| `SHA256SUMS.txt` | SHA-256 checksums for both executables. |

- Targets **x64 Windows 10 version 1507 (build 10240) or later**. Both packages run without a Python, Qt, or WebView2 installation.
- Scan with WeChat and approve sign-in. Select `+` to add a second independently authorized account. Each tab maintains its own messages and outgoing file queue.
- Pick, drag, or paste files and review the queue before sending. A batch supports up to 100 files of 1 byte to 25 MiB each. Files that fail or remain unsent stay in the queue for retry.
- Select an incoming attachment to save it. Classroom mode enlarges key controls and hides account names. Update checks link to the corresponding GitHub release.
- The 1.2 installer uses its own directory and Start menu group. A fresh launch requires QR sign-in.

### Build from source

Use x64 Go 1.26 and Inno Setup 6. Run `go test ./internal/...` from `native/`, then run `native/packaging/windows/build.ps1 -Verify` from the repository root. The first build downloads a pinned Go resource compiler. Outputs are under `native/dist/windows/`. This repository distributes the Windows application. The [separate macOS repository](https://github.com/Watertube-bilibili/wxfilehelper-macos) ships both Mac architectures. The earlier Python/Qt implementation is available at its [0.2 tag](https://github.com/Watertube-bilibili/wxfilehelper-desktop/tree/v0.2.0-beta.3).

### Privacy and licensing

Session credentials, messages, and queued file paths stay in memory. Received files are saved where you choose. WeChat processes sign-in and transfer requests. GitHub receives update checks. The service interface, account eligibility, and network conditions can affect availability. [Bilingual disclaimer](DISCLAIMER.md)

Licensed under **AGPL-3.0-only**. [License](LICENSE) · [Third-party notices](THIRD_PARTY_NOTICES.md)
