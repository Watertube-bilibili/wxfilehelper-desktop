# WX FileHelper Desktop · 微信文件传输助手桌面版

[中文](#中文) · [English](#english) · [下载 / Downloads](https://github.com/Watertube-bilibili/wxfilehelper-desktop/releases) · [免责声明 / Disclaimer](DISCLAIMER.md)

**原生 Qt 桌面应用，无内嵌网页或浏览器内核。 / Native Qt desktop app, without an embedded webpage or browser engine.**

这是独立的第三方开源项目，与腾讯、微信无隶属关系。This is an independent project, not affiliated with Tencent or WeChat.

## 中文

### 功能

- 微信扫码授权，直接通过 HTTPS 与文件传输助手服务通信。
- **双账号标签页**：点击 `+` 添加第二个微信账号。两个标签的登录、消息、草稿和文件队列各自独立，切换时继续收发。
- 原生文字收发、图片发送、附件分块上传、手动下载保存。
- 拖入或选择文件后，在待发送列表核对文件名和大小，可移除单项。发送时显示当前进度，可在当前项完成后暂停，失败与未发送的项目保留供重试。
- Fluent / UWP 风格界面。普通窗口左侧提供会话导航和下载入口，小窗口将辅助功能收进菜单。支持中英文切换、原生文件选择器和系统托盘。
- **教室大屏模式**：点击「大屏模式」或按 `F11`，进入全屏、放大字号、二维码和触控按钮，并隐藏微信昵称。按 `Esc` 或点击退出按钮恢复。
- **自动检查更新**：启动后和运行期间每 6 小时查询本仓库 GitHub Releases，也可手动检查。发现新版本后展示提示，点击进入发布页；不会静默下载或执行安装程序。
- 登录凭据与会话消息仅驻留内存。退出后重新扫码；本版不持久化聊天记录。

### 下载与使用

前往 [Releases](https://github.com/Watertube-bilibili/wxfilehelper-desktop/releases) 选择：

| 文件 | 用途 |
| --- | --- |
| `WXFileHelper-…-Windows-x64-Setup.exe` | 安装版，安装到当前用户目录，带卸载入口；无需管理员权限 |
| `WXFileHelper-…-Windows-x64-Portable.exe` | 单文件免安装版，直接运行；运行时会在系统临时目录解压运行库 |
| `SHA256SUMS.txt` | 下载文件的 SHA-256 校验值 |

1. 使用 Windows 10/11 64 位系统运行应用。
2. 用手机微信扫一扫，并在手机上确认登录。点击标签栏的 `+` 后，可以用另一个微信账号扫码。
3. 核对当前标签，输入文字或添加文件，再点击「发送」。手机端使用「文件传输助手」会话。
4. 收到附件后点击「另存为」，自行选择保存位置。
5. 课堂或共享电脑使用后，退出应用并按需清理下载文件。

`Ctrl+Enter` 发送，`Ctrl+O` 添加文件，`Ctrl+T` 添加标签，`Ctrl+Tab` 切换标签。窄窗口会收紧留白和输入区，大屏模式会放大内容。Qt 跟随系统 DPI 缩放。

### 当前边界

- **第三方预览版**。每批最多添加 **100 个文件**，每个文件最多 **25 MiB**。
- 使用的是已有开源项目中描述的服务接口，不是腾讯承诺稳定的公开开发者 API；账户和网络条件可能影响可用性。
- 此前已验证单账号真实扫码与文字同步。本版的双账号并发收发、失败保留和多种窗口尺寸已通过模拟服务和界面测试。真实双账号、文件互传和不同品牌触控大屏仍需实测。
- 图片支持 PNG/JPEG/BMP；其他文件可作为普通附件发送。暂不支持完整微信聊天功能、语音播放、表情发送或历史消息云恢复。
- 更新提示不会自动安装。当前 EXE **未签名**，Windows 可能显示发布者未知的提示。
- 界面采用 Fluent/UWP 视觉风格，程序本身是 **Qt Win32 EXE**，不是 UWP/MSIX 或微软商店应用。
- 本仓库发布 **Windows** 版。macOS 源码和发行文件见 [macOS 独立仓库](https://github.com/Watertube-bilibili/wxfilehelper-macos)。

### 从源码运行

需要 Python 3.11+；发行构建使用 Python 3.13。

```powershell
git clone https://github.com/Watertube-bilibili/wxfilehelper-desktop.git
cd wxfilehelper-desktop
python -m venv .venv
.\.venv\Scripts\python.exe -m pip install -r requirements.txt
.\.venv\Scripts\python.exe run.py
```

### 测试与构建

```powershell
.\.venv\Scripts\python.exe -m pip install -r requirements-dev.txt
.\.venv\Scripts\python.exe -m pytest -q
.\scripts\build_windows.ps1 -Iscc 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe'
```

从 [Inno Setup 官网](https://jrsoftware.org/isinfo.php) 安装编译器，并按实际路径调整 `-Iscc`。请在只安装了项目依赖的虚拟环境中构建。脚本会收集许可证、启动验证两种应用包，并在 `dist/` 生成 EXE 和校验文件。安装版保留可替换的 Qt 共享库。

### 隐私、许可和免责声明

项目不运营中转服务器、不收集遥测、不保存登录口令。微信服务会收到授权后的通信请求，GitHub 会收到更新检查请求。语言及下载目录偏好保存在本机设置中；下载文件保存在你指定的位置。

本项目以 **AGPL-3.0-only** 发布，见 [LICENSE](LICENSE)。第三方库保持其原许可，见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。

**使用前请阅读完整的[中英双语免责声明](DISCLAIMER.md)。** 软件按现状提供，接口可能变化；请遵守适用法律、微信规则和机构政策，备份重要文件，并在大屏展示前确认内容适合公开。

提交 Issue 时请移除二维码、登录票据、Cookie、聊天内容和个人文件。不要把凭据贴到公开仓库。

## English

### Features

- QR authorization with WeChat, communicating directly with File Transfer HTTPS services.
- **Two account tabs.** Click `+` to scan with a second WeChat account. Sessions, messages, drafts and file queues stay separate. Transfers keep running while you switch tabs.
- Native text messaging, image sending, chunked attachment uploads, and manual save dialogs.
- Review filenames and sizes in the queue, remove individual files and see progress for the current item. You can stop after an item and send the remaining files later.
- Fluent/UWP-inspired appearance, with a sidebar in regular windows and a compact menu in smaller ones. Includes a Chinese/English UI, native file pickers and a system tray menu.
- **Classroom mode.** Click “Large screen” or press `F11` for full screen, larger text, a larger QR code and bigger touch targets. Nicknames are hidden. Press `Esc` or use the visible exit button to return.
- **Automatic update checks:** at startup and every six hours, plus a manual check. New versions link to this repository's GitHub Release page. No silent installer downloads or execution.
- Login credentials and messages stay in process memory. Restarting requires another scan; chat history is not persisted in this release.

### Download and run

Open [Releases](https://github.com/Watertube-bilibili/wxfilehelper-desktop/releases):

| Asset | Purpose |
| --- | --- |
| `WXFileHelper-…-Windows-x64-Setup.exe` | Per-user installer with uninstall support; no administrator rights required |
| `WXFileHelper-…-Windows-x64-Portable.exe` | Single EXE; extracts bundled runtime libraries to the system temporary directory at launch |
| `SHA256SUMS.txt` | SHA-256 checksums |

Run on 64-bit Windows 10/11, scan with WeChat on your phone, and confirm sign-in. Click `+` to add the second account. Check which tab is active before sending. Use the File Transfer conversation on your phone and “Save as” to keep received files. Quit and clean up downloaded files after using a shared computer.

`Ctrl+Enter` sends, `Ctrl+O` adds files, `Ctrl+T` adds a tab and `Ctrl+Tab` switches tabs. Small windows use tighter spacing. Qt follows system DPI scaling.

### Preview limitations

- This is an **independent preview**. A batch can contain up to **100 files** of up to **25 MiB each**.
- Interfaces are informed by existing open-source clients and are not stable public developer APIs. Account policies and network conditions can affect availability.
- Real single-account QR sign-in and text synchronization were verified earlier. This release uses simulated service tests for concurrent accounts and transfer failures, plus layout tests at several window sizes. Live testing of two accounts, file transfers and physical touchscreens remains pending.
- PNG/JPEG/BMP image sending is supported; other formats can be sent as attachments. Full WeChat chat features, voice playback, outgoing stickers, and cloud history restoration are outside this release.
- Update checks do not install updates. Current EXEs are **unsigned**.
- Fluent/UWP describes the appearance. This is a **Qt Win32 application**, not UWP/MSIX or a Microsoft Store package.
- This repository publishes **Windows** builds. For macOS, see the [separate macOS repository](https://github.com/Watertube-bilibili/wxfilehelper-macos).

### Development

Use Python 3.11+ (release builds use 3.13). Create a fresh virtual environment, install `requirements.txt` and run `python run.py`. To build a release, install `requirements-dev.txt`, run `python -m pytest -q` and run `scripts/build_windows.ps1` with the path to an official Inno Setup compiler. The script collects dependency notices, tests both packaged executables at launch and writes the release assets to `dist/`.

Qt is dynamically linked and WebEngine is excluded. You can modify the application and rebuild it with replacement LGPL-covered libraries using the included build scripts.

### Privacy, license, and disclaimer

No project-operated relay, telemetry collection, or credential persistence. WeChat receives authorized service requests; GitHub receives update-check requests. Language/download-folder preferences stay in local settings, and files are saved where you choose.

Licensed under **AGPL-3.0-only**; see [LICENSE](LICENSE) and [third-party notices](THIRD_PARTY_NOTICES.md).

**Read the full [bilingual disclaimer](DISCLAIMER.md).** The software is provided as is, interfaces may change, and important files should be backed up. Follow applicable laws, service rules, and institutional policies. Review content before presenting on a shared display.

Remove QR codes, tickets, cookies, messages, and private files before submitting a public issue.
