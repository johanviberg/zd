# Multi-Step Workflows

Recipes for common multi-step operations using `zd`.

## Bulk status update

Search for tickets matching criteria, then update each one:

```bash
# Find all open tickets tagged "resolved-upstream"
zd tickets search "status:open tags:resolved-upstream" -o ndjson --fields id | \
  jq -r '.id' | \
  while read -r id; do
    zd tickets update "$id" -o json --status solved --comment "Resolved upstream" --non-interactive
  done
```

## Ticket triage

Find unassigned high-priority tickets and assign them:

```bash
# List unassigned urgent/high tickets
zd tickets search "status:new priority:urgent OR priority:high" -o json --fields id,subject,priority

# Assign a ticket
zd tickets update 12345 -o json --assignee-id 67890 --status open --add-tags triaged
```

## Daily summary report

Generate a summary of today's ticket activity:

```bash
# Open tickets by priority
zd tickets search "status:open" -o json --fields id,priority,subject --limit 100

# Tickets created today
zd tickets search "created>$(date +%Y-%m-%d) status:open" -o json --fields id,subject,priority,assignee_id

# Tickets solved today
zd tickets search "status:solved updated>$(date +%Y-%m-%d)" -o json --fields id,subject
```

## Idempotent ticket creation

Create tickets that are safe to retry (e.g., from a CI pipeline):

```bash
# First run: creates the ticket
zd tickets create -o json \
  --subject "Deploy failed: build #42" \
  --comment "Build 42 failed at step 3" \
  --priority high \
  --idempotency-key "build-failure-42" \
  --if-exists skip \
  --non-interactive

# Subsequent runs: returns existing ticket without creating a duplicate
zd tickets create -o json \
  --subject "Deploy failed: build #42" \
  --comment "Build 42 failed at step 3" \
  --priority high \
  --idempotency-key "build-failure-42" \
  --if-exists skip \
  --non-interactive
```

The `--if-exists` flag controls behavior when an idempotent ticket already exists:

| Value | Behavior |
|-------|----------|
| `error` | Fail with an error (default) |
| `skip` | Return the existing ticket unchanged |
| `update` | Update the existing ticket with new subject/comment |

## Paginated iteration

Iterate through all results using cursor-based pagination:

```bash
cursor=""
while true; do
  if [ -z "$cursor" ]; then
    result=$(zd tickets list -o json --limit 100 --status open 2>/tmp/zd-stderr)
  else
    result=$(zd tickets list -o json --limit 100 --status open --cursor "$cursor" 2>/tmp/zd-stderr)
  fi

  echo "$result"

  # Extract next cursor from stderr
  cursor=$(grep -oP 'Use --cursor "\K[^"]+' /tmp/zd-stderr 2>/dev/null || true)
  if [ -z "$cursor" ]; then
    break
  fi
done
```

## Multi-account operations

Work with multiple Zendesk accounts using profiles:

```bash
# Set up profiles
zd auth login --method token --subdomain acme --email a@acme.com --api-token TOKEN1 --profile acme
zd auth login --method token --subdomain corp --email a@corp.com --api-token TOKEN2 --profile corp

# Query both accounts
zd tickets search "status:open priority:urgent" -o json --profile acme
zd tickets search "status:open priority:urgent" -o json --profile corp
```

## Safe deletion workflow

Use the two-step confirmation to prevent accidental deletions:

```bash
# Step 1: preview what will be deleted
DRY_RUN=$(zd tickets delete 12345 -o json --dry-run)
echo "$DRY_RUN" | jq .

# Step 2: extract confirmation ID and execute
CONFIRM_ID=$(echo "$DRY_RUN" | jq -r '.confirmation_id')
zd tickets delete 12345 -o json --confirm "$CONFIRM_ID"
```

## Custom fields

Set custom field values using the field's numeric ID:

```bash
# Set a single custom field
zd tickets create -o json \
  --subject "Hardware request" \
  --comment "Need a new monitor" \
  --custom-field 360012345=monitor

# Set multiple custom fields
zd tickets update 12345 -o json \
  --custom-field 360012345=approved \
  --custom-field 360067890="2024-03-15"
```

## Tag management

```bash
# Add tags without removing existing ones
zd tickets update 12345 -o json --add-tags escalated,reviewed

# Remove specific tags
zd tickets update 12345 -o json --remove-tags spam,duplicate

# Replace all tags
zd tickets update 12345 -o json --tags billing,priority
```

## Conflict-safe updates

Use `--safe-update` to detect concurrent modifications:

```bash
zd tickets update 12345 -o json --status solved --safe-update
```

If the ticket was modified between your read and update, the request fails instead of silently overwriting changes.
