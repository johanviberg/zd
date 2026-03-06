# Zendesk Search Query Syntax

Reference for the search query language used with `zd tickets search`.

## Field operators

| Field | Values / format | Example |
|-------|----------------|---------|
| `status` | `new`, `open`, `pending`, `hold`, `solved`, `closed` | `status:open` |
| `priority` | `urgent`, `high`, `normal`, `low` | `priority:high` |
| `type` | `problem`, `incident`, `question`, `task` | `type:incident` |
| `assignee` | Agent name or email | `assignee:jane` |
| `group` | Group name | `group:billing` |
| `requester` | Requester name or email | `requester:john@acme.com` |
| `subject` | Text in subject line | `subject:"password reset"` |
| `description` | Text in first comment | `description:"login error"` |
| `tags` | Tag name | `tags:vip` |
| `organization` | Organization name | `organization:acme` |
| `created` | Date or date range | `created>2024-01-01` |
| `updated` | Date or date range | `updated>2024-06-01` |

## Boolean operators

**AND** (implicit — separate terms with spaces):

```bash
zd tickets search "status:open priority:high assignee:jane"
```

**OR** (explicit keyword):

```bash
zd tickets search "status:open OR status:pending"
```

**Negation** (prefix with `-`):

```bash
zd tickets search "-status:closed -tags:spam"
```

## Date ranges

Dates use `YYYY-MM-DD` format with comparison operators:

```bash
# Created after a date
zd tickets search "created>2024-01-01"

# Created before a date
zd tickets search "created<2024-06-01"

# Created within a range
zd tickets search "created>2024-01-01 created<2024-06-01"

# Updated in the last 7 days (relative)
zd tickets search "updated>2024-03-01"
```

## Quoting

Use double quotes for multi-word values:

```bash
zd tickets search "subject:\"password reset\" status:open"
```

## Practical examples

### Find unassigned urgent tickets

```bash
zd tickets search "status:open priority:urgent -assignee:*" -o json
```

### Find VIP tickets updated recently

```bash
zd tickets search "tags:vip updated>2024-01-01 -status:closed" -o json
```

### Find all incidents for a requester

```bash
zd tickets search "type:incident requester:john@acme.com" -o json
```

### Find open tickets in a group

```bash
zd tickets search "status:open group:support" -o json --sort-by created
```

### Find tickets by subject keyword

```bash
zd tickets search "subject:\"API error\" status:open" -o json
```

### Find tickets with specific tags

```bash
zd tickets search "tags:billing tags:escalated" -o json
```

### Find solved tickets from last month

```bash
zd tickets search "status:solved updated>2024-02-01 updated<2024-03-01" -o json
```

### Find pending tickets without a priority

```bash
zd tickets search "status:pending -priority:urgent -priority:high -priority:normal -priority:low" -o json
```

### Find all open or pending tickets

```bash
zd tickets search "status:open OR status:pending" -o json
```

### Large result sets (>1000 tickets)

The standard search endpoint returns up to 1000 results. Use `--export` for larger sets:

```bash
zd tickets search "status:closed created>2024-01-01" -o json --export
```

## Full reference

Zendesk search documentation: https://support.zendesk.com/hc/en-us/articles/4408886879258
