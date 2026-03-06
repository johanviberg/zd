# Error Handling

## Exit codes

| Code | Constant | Meaning | Recovery |
|------|----------|---------|----------|
| 0 | `ExitSuccess` | Success | — |
| 1 | `ExitGeneral` | General error | Read stderr for details |
| 2 | `ExitArgError` | Invalid argument | Fix command flags or arguments |
| 3 | `ExitAuthError` | Authentication failure | Set env vars or run `zd auth login` |
| 4 | `ExitRetryable` | Rate limited / transient error | Wait `retryAfter` seconds, then retry |
| 5 | `ExitNotFound` | Resource not found | Verify the ticket ID exists |

## Structured error format

When using `-o json`, errors are written to stderr as structured JSON:

```json
{
  "code": "rate_limited",
  "message": "Rate limited by Zendesk API",
  "exitCode": 4,
  "retryAfter": 30
}
```

Error codes map to:

| JSON `code` | HTTP status | Exit code |
|-------------|-------------|-----------|
| `auth_error` | 401, 403 | 3 |
| `not_found` | 404 | 5 |
| `rate_limited` | 429 | 4 |
| `invalid_argument` | 400 | 2 |
| `error` | 5xx, other | 1 |

## Rate limiting

The CLI has built-in retry logic for rate limits and transient errors:

- **Automatic retries**: up to 3 attempts with exponential backoff and jitter
- **Retried status codes**: 429 (rate limited) and 5xx (server errors)
- **If all retries fail**: exits with code 4 and includes `retryAfter` in the error

When the CLI exits with code 4, the caller should wait for the number of seconds in `retryAfter` before retrying.

## Auth error recovery

Exit code 3 means credentials are missing or invalid:

1. Check if env vars are set: `ZENDESK_OAUTH_TOKEN` or `ZENDESK_EMAIL` + `ZENDESK_API_TOKEN`
2. Check stored credentials: `zd auth status -o json --profile PROFILE`
3. Re-authenticate: `zd auth login --method token --subdomain SUB --email EMAIL --api-token TOKEN`

## Common error scenarios

| Scenario | Exit code | Resolution |
|----------|-----------|------------|
| No credentials configured | 3 | Run `zd auth login` or set env vars |
| Expired OAuth token | 3 | Re-run `zd auth login` to refresh |
| Ticket ID doesn't exist | 5 | Verify ID; ticket may have been deleted |
| Invalid flag value | 2 | Check `zd schema --command "..."` for valid values |
| Zendesk API outage | 4 | Wait and retry; check status.zendesk.com |
| Missing required flag | 2 | Add `--subject` and `--comment` for create |
| Subdomain not set | 2 | Use `--subdomain` flag or set in config |
| Confirmation ID mismatch | 1 | Re-run `--dry-run` to get a fresh confirmation ID |
