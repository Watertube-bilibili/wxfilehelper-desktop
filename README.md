# 超级文件 · Ultra File

[中文](#中文) · [English](#english) · [Windows 下载](https://github.com/Watertube-bilibili/wxfilehelper-desktop/releases) · [macOS 仓库](https://github.com/Watertube-bilibili/wxfilehelper-macos) · [免责声明 / Disclaimer](DISCLAIMER.md)

## 中文

2.0 预览版的传输核心使用 Go。Windows 界面采用 Win32，macOS 界面采用 AppKit。软件直接连接微信文件传输服务，通过手机微信扫码授权。本项目独立维护，与腾讯和微信无隶属关系。

每个标签右上角可「切换服务」。最多四个标签可以同时运行。更多菜单里的「当前服务设为启动默认」会保存到当前 Windows 用户。

| 服务 | 收发方法 |
| --- | --- |
| 微信文件传输助手 | 用微信扫码，在当前标签内收发。 |
| 超级文件直传 | 手机和电脑连接同一局域网。在电脑上启用直传，再用安卓客户端扫码。手机可以选择文件发送。反向传输时先在电脑选择文件，再在手机点「接收电脑上的文件」并选择保存位置。 |
| LocalSend | 启用连接后，在本标签接受收件。发送时输入对方局域网 IPv4，核对证书指纹，由对方接受。 |
| 手机 QQ「我的电脑／传输到电脑」 | 暂未找到允许第三方作为个人电脑端收发的公开接口，入口暂不可用。 |

直传以 4 MiB 分块，通过两路连接交错传输，失败的块最多重试两次，完成后校验整份文件的 SHA-256。单文件上限 8 GiB。配对码含当前会话的密钥，接收者可以下载当前共享文件并上传文件。电脑收件在下载目录的 Ultra File 子目录。关闭连接所在的标签或退出程序会终止配对。

### Windows 下载与使用

| 文件 | 用途 |
| --- | --- |
| `UltraFile-2.0.0-Windows-x64-Portable.exe` | 单文件，双击运行 |
| `UltraFile-2.0.0-Windows-x64-Setup.exe` | 当前用户安装，提供开始菜单和卸载入口 |
| `SHA256SUMS.txt` | 核对两个 EXE 的 SHA-256 |

- 兼容目标为 **Windows 10 1507（内部版本 10240）及以上的 x64 系统**。两个包都无需另装 Python、Qt 或 WebView2。
- 用手机微信扫一扫，确认授权，再在当前标签发送文字或文件。点击 `+` 可添加第二个账号。两个标签各自保留消息和待发送文件。
- 选择、拖入或粘贴文件，核对发送列表后点击「发送」。每批最多 100 个文件，每项 1 字节到 25 MiB。发送失败时未完成的文件仍可重试。
- 手机发送 PPTX、Word 文档、PDF 或其他普通附件后，可在会话中选中文件，点击「另存为」。如果微信只提供文件名，仍然会显示附件。
- 关闭主窗口会收起到 Windows 通知区域，已登录的账号继续收件。后台收到附件时，点击系统通知可回到对应账号。要结束会话，右键托盘图标，选择「退出程序」。
- 安装时可选择「创建桌面快捷方式」和「登录 Windows 时在后台启动」。自启动默认关闭。重启程序后需重新扫码。
- 程序在启动时和运行期间每 6 小时检查 GitHub 更新，自动检查失败后 15 分钟重试。发现新版时，窗口内会出现发布页入口。

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

### 安卓构建 / Android build

Android 7.0+。使用 JDK 17、Android SDK Platform 35、Build Tools 35、ZXing core 3.5.3 和自己的签名密钥。Set ULTRAFILE_STORE_PASSWORD locally and run android/build.ps1 with -Jdk, -BuildTools, -AndroidJar, -Zxing, -KeyStore and -Alias. The signed APK is written to android/dist/.

## English

The 2.0 preview uses a shared Go transfer core with Win32 on Windows and AppKit on macOS. It connects directly to WeChat File Transfer, with sign-in approved by scanning a QR code in WeChat. This independently maintained project is unaffiliated with Tencent and WeChat.

Use **Services** at the top of a tab. Run up to four tabs concurrently and use the menu to save a startup service for the current Windows user.

| Service | How to transfer |
| --- | --- |
| WeChat File Transfer | Scan with WeChat and use the tab's conversation. |
| Ultra File LAN | Join the same LAN, enable the desktop listener and scan its QR in the Android companion. Pick a file on Android to upload. To download, first share a file on the PC, then choose Receive and a destination on the phone. |
| LocalSend | Accept incoming HTTPS transfers in the tab. To send, enter the receiver’s LAN IPv4, confirm its certificate fingerprint and wait for approval. |
| Mobile QQ – My Computer | Currently unavailable. We have not identified a public API for registering a third-party app as the user's personal computer. |

Two lanes carry alternating 4 MiB chunks, retry failures up to twice, and verify the final SHA-256. Each file can be up to 8 GiB. Pairing credentials allow the holder to upload and download the shared file. Windows saves incoming LAN files under Downloads/Ultra File. Closing the associated tab or quitting ends pairing.

### Download and use on Windows

| Asset | Use |
| --- | --- |
| `UltraFile-2.0.0-Windows-x64-Portable.exe` | Single executable. Double-click to launch. |
| `UltraFile-2.0.0-Windows-x64-Setup.exe` | Per-user installation with a Start menu and uninstaller. |
| `SHA256SUMS.txt` | SHA-256 checksums for both executables. |

- Targets **x64 Windows 10 version 1507 (build 10240) or later**. Both packages run without a Python, Qt, or WebView2 installation.
- Scan with WeChat and approve sign-in. Select `+` to add a second independently authorized account. Each tab maintains its own messages and outgoing file queue.
- Pick, drag, or paste files and review the queue before sending. A batch supports up to 100 files of 1 byte to 25 MiB each. Files that fail or remain unsent stay in the queue for retry.
- Select a presentation, document, PDF or other attachment from your phone and choose **Save as**. A file still appears by name if WeChat omits the download details.
- Closing the window keeps signed-in accounts receiving files in the Windows notification area. Click a background file notification to open the corresponding account. Right-click the tray icon and choose **Quit Ultra File** to end the session.
- Setup offers a desktop shortcut and optional launch in the background at Windows sign-in. Automatic startup is off by default. Scan again after restarting the app.
- The app checks GitHub at startup and every six hours while running. Failed scheduled checks retry after 15 minutes. An available update appears in the window with a link to its release.

### Build from source

Use x64 Go 1.26 and Inno Setup 6. Run `go test ./internal/...` from `native/`, then run `native/packaging/windows/build.ps1 -Verify` from the repository root. The first build downloads a pinned Go resource compiler. Outputs are under `native/dist/windows/`. This repository distributes the Windows application. The [separate macOS repository](https://github.com/Watertube-bilibili/wxfilehelper-macos) ships both Mac architectures. The earlier Python/Qt implementation is available at its [0.2 tag](https://github.com/Watertube-bilibili/wxfilehelper-desktop/tree/v0.2.0-beta.3).

### Privacy and licensing

Session credentials, messages, and queued file paths stay in memory. Received files are saved where you choose. WeChat processes sign-in and transfer requests. GitHub receives update checks. The service interface, account eligibility, and network conditions can affect availability. [Bilingual disclaimer](DISCLAIMER.md)

Licensed under **AGPL-3.0-only**. [License](LICENSE) · [Third-party notices](THIRD_PARTY_NOTICES.md)
