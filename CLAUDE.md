# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
go build -o zd                        # Build binary
go test ./...                          # Run all tests
go test ./internal/api/ -run TestName  # Run a single test
go vet ./...                           # Lint
```

No external test dependencies — tests use `net/http/httptest` with JSON fixtures in `testdata/`.

## Architecture

Go CLI (`zd`) for Zendesk's ticketing REST API. Module: `github.com/johanviberg/zd`. Uses Cobra + Viper + XDG.

### Request flow

Commands in `cmd/` are thin wiring: they read flags, call a service method, and format output. The core flow is:

1. `rootCmd.PersistentPreRunE` loads config and creates a `Formatter`, both stored in `context.Context` via typed keys
2. Command handlers retrieve these via `configFromCtx()` / `formatterFromCtx()`
3. `newTicketService(cmd)` / `newSearchService(cmd)` resolve credentials, build an API client, and return a service
4. Service methods call `client.doJSON()` which handles HTTP + JSON decoding
5. Formatter outputs results to stdout; errors go to stderr

### HTTP transport chain

`api.NewClient` builds a layered `http.RoundTripper`:

```
RetryTransport (exponential backoff + jitter for 429/5xx, max 3 retries)
  → AuthTransport (adds Basic or Bearer header from ProfileCredentials)
    → http.Transport (TLS 1.2+)
```

### Error handling

`types.AppError` carries a `Code`, `Message`, and `ExitCode` (0-5). API status codes map to specific AppError types: 401/403→AuthError, 404→NotFoundError, 429→RetryableError. `output.ExitWithError` renders errors (structured JSON when `--output json`) and exits with the appropriate code.

### Auth resolution order

`auth.ResolveCredentials`: env vars (`ZENDESK_OAUTH_TOKEN` or `ZENDESK_EMAIL`+`ZENDESK_API_TOKEN`) → stored credentials file (`$XDG_CONFIG_HOME/zd/credentials.json`) by profile.

### Config

`config.ConfigDir()` reads `XDG_CONFIG_HOME` env directly (not the `adrg/xdg` cached value) for test compatibility. Config is per-profile under `profiles.<name>` in `config.yaml`.

### Key packages

- `cmd/` — Cobra commands. Each subcommand file registers itself via `init()`. `tickets.go` has shared `newTicketService`/`newSearchService` factory functions.
- `internal/api/` — `Client` (HTTP), `TicketService`, `SearchService`, `RetryTransport`, cursor-based pagination.
- `internal/auth/` — `AuthTransport` RoundTripper, credential CRUD (`credentials.json`), `OAuthFlow` (browser-based).
- `internal/output/` — `Formatter` interface with JSON/NDJSON/Text implementations, field projection via `projectFields`.
- `internal/types/` — Domain types (`Ticket`, `TicketPage`, `SearchPage`), `AppError`, pagination options.
- `pkg/zendesk/` — Public interfaces (`TicketService`, `SearchService`) used by commands.

### Test pattern

Tests create an `httptest.Server` with inline handlers or fixture data, then construct a `Client` pointing at `server.URL` (bypassing real auth/transport). See `testClient()` helper in `internal/api/client_test.go`.

## Agent discovery

- `zd commands -o json` — full CLI surface with flags/types/defaults
- `zd schema --command "tickets create"` — JSON Schema for tool calling

## Output

- `--output json|ndjson|text` (default: `text`)
- `--fields id,status,subject` — field projection
- Errors always to stderr; structured JSON errors when `--output json`

## Exit codes

0=success, 1=general, 2=arg error, 3=auth, 4=retryable/rate-limited, 5=not found
