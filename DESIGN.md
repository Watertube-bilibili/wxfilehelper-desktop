# Desktop design

The user selected a Fluent/UWP-inspired native utility with classroom display support and asked to proceed directly.
Build with real Qt Widgets, not a web mockup. This session is code-first.

- Native system title bar, menus, file pickers and keyboard conventions.
- Windows Fluent neutral surfaces, pale gray utility rail, dark readable text.
- Windows blue primary actions and pale blue sent-message bubbles.
- System UI fonts for legibility in Chinese and English; 14px body, 22px headings.
- A compact left rail holds the one conversation, downloads and project information.
- The main area transitions from QR authorization to a chat stream and composer.
- Native progress bars show active transfers. Failure keeps recoverable input.
- No sample messages in normal operation, fake online indicators or fabricated success.
- Classroom mode uses the screen's available height to select text and QR sizes,
  hides the utility rail and nickname, and makes primary touch targets at least 60px tall.
- F11 enters full screen; Esc and a visible header button exit it. Touch scroll stays native.
- Automatic updates show an unobtrusive banner and open only this repository's release pages.
- Resize down to 760×600; wrap long filenames and message text.
- Icons are original simple line geometry; no copied Tencent assets.

## Surface contract

Mode: operate. Primary task: send a file to one's own phone and retrieve a received file.
First viewport: recognizable transfer identity, honest connection state, central QR,
short scan instructions, refresh action, unobtrusive open-source attribution.
Signature interaction: drop local files into the composer to stage them; explicitly
send, then see upload progress change into a delivered-to-service message row.
States: starting, awaiting scan, awaiting phone confirmation, online, reconnecting,
expired, failed; outgoing messages are added only after server acknowledgement.
Risk: upstream interfaces are undocumented; initial release must disclose missing
account-level testing and unsigned binaries.
