# v0.2.0-beta.1 · Windows 双账号预览版 / Dual-account Windows preview

## 中文

- **同时登录两个微信账号。** 点 `+` 添加第二个账号并扫码。标签之间的登录、消息、草稿和文件列表独立，切换时传输继续。
- **批量收发文件。** 拖入文件或点击「添加文件」，即可核对名称和大小、移除单项。发送时显示当前文件进度，可在当前项完成后停止。失败与未发送的项目保留在列表中。
- **小窗和教室大屏。** 窄窗口收紧间距，宽屏限制消息栏宽度。按 `F11` 放大文字、二维码和触控目标并隐藏昵称，按 `Esc` 返回。
- **打包启动修复。** 移除构建时误带的冲突 DLL，对安装版与单文件版做了启动验证。应用会自动检查此仓库的 GitHub Releases。

在 Assets 中选 `Setup.exe` 安装，或运行 `Portable.exe`。使用 `SHA256SUMS.txt` 核对下载。适用于 Windows 10/11 x64。预览包未签名，每个文件最多 25 MiB。

已用模拟服务验证两个账号并发传输、失败保留和不同窗口尺寸。此前验证过真实单账号扫码与文字同步，真实双账号附件互传和教室触控设备仍待实测。项目采用 AGPL-3.0-only，与腾讯无隶属关系。[双语免责声明](https://github.com/Watertube-bilibili/wxfilehelper-desktop/blob/main/DISCLAIMER.md)

## English

- **Sign in to two WeChat accounts at once.** Click `+` and scan with the second account. Each tab keeps its own sign-in, messages, draft and queue. Transfers continue when you switch tabs.
- **Batch file transfers.** Drop files or use Add files to review names and sizes and remove individual items. See the active file's progress, stop after the current item, and keep failed or unsent items for retry.
- **Compact and classroom layouts.** Small windows use tighter spacing, and wide displays keep the conversation at a readable width. `F11` enlarges text, the QR code and touch targets and hides nicknames. `Esc` exits full screen.
- **Packaged startup fixed.** Rebuilt without a conflicting DLL and checked startup of both Windows packages. The app checks this repository's GitHub Releases for updates.

Choose `Setup.exe` or run `Portable.exe` under Assets. Verify downloads with `SHA256SUMS.txt`. For Windows 10/11 x64. The preview binaries are unsigned. Each file may be up to 25 MiB.

Simulated service tests covered concurrent accounts, failed transfers and multiple window sizes. Live single-account QR sign-in and text sync were verified earlier. Live attachment transfers with two accounts and physical classroom touchscreens remain unverified. This independent AGPL-3.0-only project is unaffiliated with Tencent. [Bilingual disclaimer](https://github.com/Watertube-bilibili/wxfilehelper-desktop/blob/main/DISCLAIMER.md)
