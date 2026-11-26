# 超级文件 2.0.0 预览版 · Ultra File 2.0.0 preview

每个标签右上角可「切换服务」。最多四个标签可以同时运行。更多菜单里的「当前服务设为启动默认」会保存到当前 Windows 用户。

| 服务 | 收发方法 |
| --- | --- |
| 微信文件传输助手 | 用微信扫码，在当前标签内收发。 |
| 超级文件直传 | 手机和电脑连接同一局域网。在电脑上启用直传，再用安卓客户端扫码。手机可以选择文件发送。反向传输时先在电脑选择文件，再在手机点「接收电脑上的文件」并选择保存位置。 |
| LocalSend | 启用连接后，在本标签接受收件。发送时输入对方局域网 IPv4，核对证书指纹，由对方接受。 |
| 手机 QQ「我的电脑／传输到电脑」 | 暂未找到允许第三方作为个人电脑端收发的公开接口，入口暂不可用。 |

直传以 4 MiB 分块，通过两路连接交错传输，失败的块最多重试两次，完成后校验整份文件的 SHA-256。单文件上限 8 GiB。配对码含当前会话的密钥，接收者可以下载当前共享文件并上传文件。电脑收件在下载目录的 Ultra File 子目录。关闭连接所在的标签或退出程序会终止配对。

Use **Services** at the top of a tab. Run up to four tabs concurrently and use the menu to save a startup service for the current Windows user.

| Service | How to transfer |
| --- | --- |
| WeChat File Transfer | Scan with WeChat and use the tab's conversation. |
| Ultra File LAN | Join the same LAN, enable the desktop listener and scan its QR in the Android companion. Pick a file on Android to upload. To download, first share a file on the PC, then choose Receive and a destination on the phone. |
| LocalSend | Accept incoming HTTPS transfers in the tab. To send, enter the receiver’s LAN IPv4, confirm its certificate fingerprint and wait for approval. |
| Mobile QQ – My Computer | Currently unavailable. We have not identified a public API for registering a third-party app as the user's personal computer. |

Two lanes carry alternating 4 MiB chunks, retry failures up to twice, and verify the final SHA-256. Each file can be up to 8 GiB. Pairing credentials allow the holder to upload and download the shared file. Windows saves incoming LAN files under Downloads/Ultra File. Closing the associated tab or quitting ends pairing.

兼容目标为 x64 Windows 10 1507 至 Windows 11。所有系统版本和账户组合尚待实机验证。Targets x64 Windows 10 build 10240 and later. Full device and account coverage is pending.

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
