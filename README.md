# Zendesk CLI (`zd`)

An unofficial, agent-friendly command-line interface for Zendesk's ticketing REST API. Built for both humans and AI agents.

## What it does

`zd` lets you manage Zendesk tickets from the terminal. List, search, create, update, and delete tickets with structured output that works well in scripts and AI agent workflows. It includes built-in discovery commands (`zd commands` and `zd schema`) that let AI agents introspect the CLI at runtime, so they can figure out what's available without hardcoded knowledge.

## Installation

### Homebrew

```bash
brew install johanviberg/tap/zd
```

### go install

```bash
go install github.com/johanviberg/zd@latest
```

### Build from source

```bash
git clone https://github.com/johanviberg/zd.git
cd zd
go build -o zd
```

## Authentication

Choose **one** of the methods below. Do not mix methods — `zd` uses the first credentials it finds (environment variables take priority over stored credentials).

### OAuth (recommended)

OAuth is the recommended authentication method. It uses a browser-based consent flow, avoids putting secrets on the command line, and the resulting token is scoped to only the permissions you grant.

#### 1. Register an OAuth client in Zendesk

1. In Admin Center, go to **Apps and integrations → APIs → OAuth clients**, then click **Add OAuth client**.
2. Fill in the fields:
   - **Name** — e.g. `zd CLI` (shown to users on the consent screen)
   - **Description** — optional
   - **Client kind** — select **Confidential** (the CLI runs locally and stores the secret with restricted file permissions)
   - **Redirect URLs** — enter `http://127.0.0.1/callback` (the CLI starts a local server on a random port; Zendesk matches on host and path, ignoring the port for localhost)
3. Click **Save**. A **Secret** field appears — copy it immediately, it is only shown in full once.
4. Note the **Identifier** field — this is the Client ID.

#### 2. Log in

```bash
zd auth login \
  --subdomain mycompany \
  --client-id YOUR_CLIENT_ID \
  --client-secret YOUR_CLIENT_SECRET
```

This opens a browser window for the OAuth consent flow. The CLI requests `read write` scopes. The token is stored locally.

### API token

If you can't use OAuth (e.g. in headless environments or CI), you can authenticate with an API token instead:

```bash
zd auth login --method token \
  --subdomain mycompany \
  --email you@example.com \
  --api-token YOUR_API_TOKEN
```

### Environment variables

As an alternative to `auth login`, you can set environment variables. This is useful for CI/CD or scripts. Environment variables always take priority over stored credentials.

```bash
# OAuth token
export ZENDESK_SUBDOMAIN=mycompany
export ZENDESK_OAUTH_TOKEN=your_oauth_token

# Or API token
export ZENDESK_SUBDOMAIN=mycompany
export ZENDESK_EMAIL=you@example.com
export ZENDESK_API_TOKEN=your_token
```

### Check auth status

```bash
zd auth status
```

## Quick start

```bash
# List recent tickets
zd tickets list

# Show a specific ticket
zd tickets show 12345

# Create a ticket
zd tickets create --subject "Printer broken" --comment "The office printer is not responding"

# Update a ticket
zd tickets update 12345 --status pending --comment "Waiting on vendor" --public=false

# Search tickets
zd tickets search "status:open priority:high"

# Delete a ticket (requires confirmation)
zd tickets delete 12345 --yes
```

## Output formats

Use `--output` (or `-o`) to control how results are formatted:

```bash
# Human-readable table (default)
zd tickets list

# JSON
zd tickets list -o json

# Newline-delimited JSON (one object per line, good for piping)
zd tickets list -o ndjson
```

### Field projection

Use `--fields` to select specific fields:

```bash
zd tickets list --fields id,status,subject -o json
```

### Sideloading related records

Use `--include` to sideload related data (e.g. users) alongside tickets. This resolves IDs like `requester_id` and `assignee_id` into human-readable names and emails:

```bash
# Show a ticket with requester and assignee names
zd tickets show 12345 --include users

# List tickets with user names in the table
zd tickets list --include users

# Combine with field projection
zd tickets show 12345 --include users --fields id,subject,requester_name,assignee_name
```

When users are sideloaded, the output is enriched with `requester_name`, `requester_email`, `assignee_name`, and `assignee_email` fields.

Errors always go to stderr. When using `--output json`, errors are also structured JSON on stderr.

## Using with AI agents

`zd` is designed to be used by AI agents like Claude Code. The quickest way to get started is to install the bundled agent skill.

### Agent skill

`zd` ships with an [agent skill](https://skills.sh/) that teaches your AI agent how to authenticate, discover commands, handle errors, and run common workflows. Install it with the [skills CLI](https://github.com/vercel-labs/skills):

```bash
npx skills add johanviberg/zd
```

This copies the skill into your agent's skills directory (e.g. `.claude/skills/` for Claude Code). Once installed, the agent can use `zd` without any additional setup in your project files.

### Self-describing commands

Two built-in commands make `zd` discoverable at runtime, even without the skill:

#### Command discovery

`zd commands` lists every available command with its flags, types, defaults, and argument names:

```bash
zd commands -o json
```

An agent can call this once to learn the full CLI surface.

#### JSON Schema for tool calling

`zd schema` generates a JSON Schema for any command's input, which maps directly to tool-calling conventions:

```bash
zd schema --command "tickets create"
```

This returns a schema with property types, required fields, and defaults that an agent can use to construct valid calls.

### Example: Claude Code with CLAUDE.md

Add something like this to your project's `CLAUDE.md` to give Claude Code access to Zendesk:

```markdown
## Zendesk

Use the `zd` CLI to interact with Zendesk. Auth is already configured.

- Run `zd commands -o json` to discover available commands
- Run `zd schema --command "<command>"` to get the input schema for a command
- Always use `--output json` when reading ticket data
- Use `--non-interactive` to prevent prompts
```

### Example: MCP tool definition

You can wrap `zd` as an MCP tool by pointing your server at the binary and using `zd schema` to generate input schemas dynamically.

## Command reference

| Command | Description |
|---|---|
| `zd auth login` | Authenticate with Zendesk (OAuth or API token) |
| `zd auth logout` | Remove stored credentials |
| `zd auth status` | Show current authentication status |
| `zd tickets list` | List tickets (supports `--include`) |
| `zd tickets show <id>` | Show a ticket (supports `--include`) |
| `zd tickets create` | Create a ticket |
| `zd tickets update <id>` | Update a ticket |
| `zd tickets delete <id>` | Delete a ticket |
| `zd tickets search <query>` | Search tickets (supports `--include`) |
| `zd config show` | Show current configuration |
| `zd config set <key> <value>` | Set a configuration value |
| `zd commands` | List all commands with flags (for agent discovery) |
| `zd schema --command "..."` | JSON Schema for a command's input |
| `zd version` | Print version information |

### Global flags

| Flag | Description |
|---|---|
| `-o, --output` | Output format: `text`, `json`, `ndjson` (default: `text`) |
| `--fields` | Field projection (comma-separated) |
| `--no-headers` | Omit table headers in text mode |
| `--non-interactive` | Never prompt for input |
| `--yes` | Auto-confirm prompts |
| `--subdomain` | Override Zendesk subdomain |
| `--profile` | Config profile (default: `default`) |
| `--debug` | Debug logging to stderr |
| `--trace-id` | Trace ID attached to API requests |

## Configuration

Config files live in `$XDG_CONFIG_HOME/zd/` (typically `~/.config/zd/`):

- `config.yaml` -- settings per profile
- `credentials.json` -- stored auth tokens (file permissions: 0600)

### Profiles

You can maintain multiple Zendesk accounts using profiles:

```bash
# Login to a second account
zd auth login --profile staging --subdomain mycompany-staging --method token \
  --email you@example.com --api-token STAGING_TOKEN

# Use it
zd tickets list --profile staging
```

### Setting config values

```bash
zd config set subdomain mycompany
zd config show
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error |
| 2 | Argument error |
| 3 | Authentication error |
| 4 | Retryable error (rate limited) |
| 5 | Not found |

## License

MIT
