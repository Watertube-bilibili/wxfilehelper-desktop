# 1.2.4 · 手机附件与后台收件 / Phone attachments and background transfers

## 中文

- 安装器支持含中文等字符的路径。覆盖安装可以取消原来的桌面快捷方式。若你后来把这个快捷方式改为指向其他程序，卸载时会保留它。
- 修复手机发送的 PPTX 等普通附件在消息正文为空或经过转义时可能被误判为“不支持”的问题。文件卡片读取微信提供的文件名、大小和下载信息。选中文件，点击「另存为」保存。
- 关闭窗口后，已登录的账号继续在托盘运行。后台收到新附件时显示 Windows 系统通知，点击即可返回对应账号。托盘右键菜单可退出程序。
- 安装器可选择创建桌面快捷方式，也可选择登录 Windows 时在后台启动。自动启动默认关闭，重启程序后仍需扫码。
- 运行期间每 6 小时检查更新，自动检查失败后 15 分钟重试。教室大屏模式及其 F11 入口已移除。
- 两个 EXE 均为 x64，兼容目标为 Windows 10 1507 及以上。附件回归使用模拟微信响应，校验收到的文件字节。此版本尚未在 1507 真机上完成手机账号端到端测试。

## English

- Setup handles Unicode installation paths and removes its desktop shortcut when deselected during an upgrade. If you repoint the shortcut to another application, uninstall now preserves it.
- Fixed incoming presentations and other files being shown as unsupported when WeChat supplied attachment metadata outside the message body or escaped the XML. Select the named attachment and choose **Save as**.
- Closing the window keeps signed-in accounts running in the notification area. Click a Windows notification for a new background attachment to open its account. Use the tray menu to quit.
- Setup offers a desktop shortcut and optional background launch at Windows sign-in. Automatic startup is off by default. A fresh process requires QR authorization.
- The app checks for updates every six hours and retries failed scheduled checks after 15 minutes. Classroom mode and its F11 shortcut have been removed.
- Both EXEs target x64 Windows 10 version 1507 and later. Attachment tests use simulated service responses and compare the saved bytes. A live phone-to-PC transfer on a physical 1507 machine remains unverified.

[免责声明 / Disclaimer](https://github.com/Watertube-bilibili/wxfilehelper-desktop/blob/main/DISCLAIMER.md)
