# Windows desktop design

Mode: operate. The user chose a light Fluent desktop utility with the classic navigation sidebar. The first task is to transfer a file to the user's own phone or save an incoming attachment. Version 1.2.1 uses the operating system's Win32 controls on the original Go architecture.

- Structure: a 196-DIP navigation rail on regular windows, a white title and account strip, and a flexible conversation canvas. Below 880 DIPs, secondary destinations move into the More actions menu. Account tabs and attachment actions remain reachable.
- Palette: cool off-white canvas `#F9FBFD`, white input and header, mist-blue rail `#EDF2F7`, ink `#1E2D3E`, muted ink `#4E6176`, Fluent action blue `#005CA5`, and powder-blue outgoing messages `#DCECFB`. Blue identifies the current account, selection and the primary action. Success, pending and failures have labels as well as color.
- Typography: Segoe UI and the system CJK fallback. Standard body uses 15 DIPs, metadata 13, headings 23; classroom body uses 21. Line icons are drawn in GDI beside labels where they help scanning. The original project icon identifies the app.
- The selected account uses a short blue indicator. Account names have limited visual width and the underlying tab remains a native keyboard and accessibility control.
- Incoming messages align left on a cool neutral fill; outgoing messages align right in powder blue. Attachments lead with the filename and expose source, time and size below it. The selected attachment can be saved with a native dialog. Long content still has its full accessible label and can be opened or copied.
- Keep the queue together with the editor and Send. Pending files retain their name and size after a failed transfer. Use a thin outline to group the composer and show keyboard focus.
- Sign-in uses a high-contrast, white-backed QR code and one recovery action. A smaller window stacks the sign-in steps vertically. Classroom mode collapses navigation, hides names, enlarges targets and places queued files next to the editor when there is room.
- Use the system title bar, native context menus, focus conventions, and open/save dialogs. F11 toggles classroom mode; Escape and the visible button exit it. Login, connectivity, transfer progress and update states reflect the session controller.
