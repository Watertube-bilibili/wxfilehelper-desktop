# Desktop design

The user selected a Fluent/UWP-inspired native utility with classroom display support and asked to proceed directly.
Build with real Qt Widgets, not a web mockup. This session is code-first.

- Native system title bar, menus, file pickers and keyboard conventions.
- Windows Fluent surfaces, a pale blue-gray navigation rail, and dark readable text.
- Windows blue primary actions and pale blue sent-message bubbles.
- System UI fonts for legibility in Chinese and English; 14px body, 22px headings.
- The sidebar restores the original conversation and files navigation, with downloads, update checks and About below. Account tabs sit at the top of the conversation. Small windows move secondary actions into a menu and keep the file filter in view.
- The main area transitions from QR authorization to a chat stream and composer.
- The queue shows file names and sizes before sending. Each tab has its own progress bar and inline failures. A batch can stop after the current item.
- No sample messages in normal operation, fake online indicators or fabricated success.
- Classroom mode uses the screen's available height to select text and QR sizes,
  hides the sidebar and nicknames, and increases the size of touch targets.
- F11 enters full screen; Esc and a visible header button exit it. Touch scroll stays native.
- Automatic updates show an unobtrusive banner and open only this repository's release pages.
- Use compact spacing down to a 590×430 logical-pixel window and cap chat width on large displays. Long filenames stay readable without pushing actions out of view.
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
