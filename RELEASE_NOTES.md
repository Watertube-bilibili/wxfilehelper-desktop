# v0.1.0-beta.1 · Windows 原生预览版 / Native Windows preview

## 中文

首个原生 Qt 版本：不嵌入网页、不包含 Chromium、不依赖已安装的微信进程。

- 微信扫码登录，文字同步与发送，图片及文件上传、下载保存。
- Fluent/UWP 风格界面，中英文切换。
- 教室大屏模式：F11 全屏、大字号、大二维码与触控按钮；Esc 退出。
- 自动检查本仓库 GitHub Releases，发现更新后提供发布页入口。
- 两种 Windows x64 分发：安装版 `Setup.exe`、单文件免安装版 `Portable.exe`。
- AGPL-3.0-only 开源；附双语 README、免责声明与第三方许可证。

**验证情况：** 自动化测试覆盖协议、安全边界、上传分块、下载中断、版本比较和界面状态。已观察到真实扫码登录和文字同步成功。全量账户/文件/触控硬件兼容性尚未完成。

**使用须知：** Windows 10/11 x64；单文件上限暂为 25 MiB。此版本未签名。它是第三方预览客户端，与腾讯无隶属关系，接口可能变化，请备份重要文件并遵守微信规则。大屏会显示消息与文件名，请在课堂展示前检查内容。详见 [免责声明](https://github.com/Watertube-bilibili/wxfilehelper-desktop/blob/main/DISCLAIMER.md)。

选择 `Setup.exe` 安装，或直接运行 `Portable.exe`。校验值见 `SHA256SUMS.txt`。macOS 将在独立仓库发布。

## English

First native Qt preview: no embedded webpage, Chromium, or dependency on an installed WeChat process.

- WeChat QR sign-in, text synchronization/sending, image/file upload and manual downloads.
- Fluent/UWP-inspired UI with Chinese/English switching.
- Classroom mode: F11 full screen, larger text/QR and touch targets; Esc exits.
- Automatic GitHub Release checks with a link to the new release.
- Windows x64 installer (`Setup.exe`) and single-file portable EXE (`Portable.exe`).
- AGPL-3.0-only source, bilingual README/disclaimer, and third-party license notices.

**Validation:** automated tests cover protocol behavior, security boundaries, chunk uploads, interrupted downloads, update comparisons, and UI states. Real QR sign-in and text synchronization were observed working. Comprehensive account/media/touch-hardware validation is not complete.

**Notes:** Windows 10/11 x64; a conservative 25 MiB per-file limit. Binaries are unsigned. This independent preview is not affiliated with Tencent. Interfaces may change; back up important files and follow service rules. Classroom mode displays message contents and filenames. Read the [disclaimer](https://github.com/Watertube-bilibili/wxfilehelper-desktop/blob/main/DISCLAIMER.md).

Use the installer or run the portable EXE. Verify downloads with `SHA256SUMS.txt`. macOS releases will use a separate repository.
