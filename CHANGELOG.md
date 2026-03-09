# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Use valid go version format in go.mod

## [0.4.2] - 2026-03-08

### Added

- Command palette with Ctrl+P
- Toggleable tags column
- Build version in footer next to email
- Linux deb/rpm packages via nfpm

### Fixed

- Newest comments shown first in detail views
- Chart anchored to bottom with improved spacing
- Equal-width panels in split view

## [0.4.1] - 2026-03-08

### Added

- Window title
- Bell notifications
- Ticket status chart in TUI

## [0.4.0] - 2026-03-07

### Added

- Built-in MCP server (`zd mcp serve`)

### Security

- Hardened auth, HTTP transport, credentials, and error handling

## [0.3.0] - 2026-03-07

### Added

- Natural language to Zendesk search query translation
- Split-panel TUI with detail side panel
- Infinite scroll pagination
- Go-to-ticket shortcut (g)
- `--demo` flag for offline showcase
- User status bar and open-in-browser
- Ticket comments and Help Center articles commands
- Hour-level time support in NLQ

### Fixed

- Clear detail panel on empty results
- Nil map panic
- Export search 422 error

## [0.2.0] - 2026-03-07

### Added

- Interactive TUI mode via `zd tui`
- Auto-refresh with countdown and manual refresh
- User sideloading to ticket commands

## [0.1.0] - 2026-03-06

### Added

- Initial release — Zendesk CLI with ticket CRUD, search, auth (OAuth + API token), JSON/text/NDJSON output, field projection, retry with backoff, and profile support

[Unreleased]: https://github.com/johanviberg/zd/compare/v0.4.2...HEAD
[0.4.2]: https://github.com/johanviberg/zd/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/johanviberg/zd/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/johanviberg/zd/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/johanviberg/zd/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/johanviberg/zd/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/johanviberg/zd/releases/tag/v0.1.0
