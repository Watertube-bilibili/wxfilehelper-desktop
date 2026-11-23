# Product

<!-- impeccable:product-schema 1 -->

## Platform

adaptive

Native desktop application for Windows and macOS; no embedded website or browser engine.

## Stack

Delegated by the user: Python and Qt Widgets (PySide6 Essentials), PyInstaller,
Windows installer, and GitHub Actions for macOS packaging.

## Users

People exchanging text and files between their own phone's WeChat File Transfer
conversation and a standalone desktop window.

## Product Purpose

Offer a focused, fully open-source native file-transfer client.

## Capabilities and Constraints

- Public GitHub repository; AGPL-3.0 license.
- Windows installer EXE and one-file portable EXE; attempt macOS distribution.
- Bilingual Chinese/English README and release notes.
- Bilingual disclaimer. Automatic update checks. Fluent/UWP appearance and classroom display mode.
- Two independently authorized account tabs, each with its own background transfers and pending queue.
- Support compact layouts and large displays with the same native controls.
- Windows release first; macOS source and releases belong in a separate public repository.
- User explicitly rejected an embedded webpage and reverse-engineering tools.
- Use normal QR authorization and publicly available open-source protocol references.
- Do not inject into, inspect memory of, or depend on an installed WeChat process.
- Account availability and undocumented service behavior are controlled by Tencent.
- A successful build does not establish account-specific transfer compatibility.

## Brand Commitments

User accepted a simple, familiar WeChat-style chat window. This is a third-party
project, not an official Tencent application; do not use Tencent logos.

## Product Principles

- Keep the transfer itself central.
- Clearly distinguish connected, pending, failed, and expired states.
- Never persist login secrets or silently send files.
- Publish reproducible source and honest validation results.
