# gh-rate-limit

A GitHub CLI extension to display your GitHub API rate limit status with visual progress bars.

## Installation

```bash
gh extension install sethrylan/gh-rate-limit
```

## Usage

```bash
# Show authenticated rate limits
gh rate-limit

# Show unauthenticated rate limits
gh rate-limit --anonymous

# Show absolute reset times instead of relative
gh rate-limit --absolute

# Output raw JSON from the GitHub API
gh rate-limit --json
```

## Example Output

```
core                         4975/5000   ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  resets in 45m 30s
graphql                      4500/5000   ████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░  resets in 32m 15s
search                         15/30     ██████████████████████████░░░░░░░░░░░░░░  resets in 45s
code_search                     0/10     ████████████████████████████████████████  resets in 1m 0s
```

- Progress bars show percentage **used** (filled portion)
- Colors indicate remaining capacity: green (>50%), yellow/orange (20-50%), red (<20%)
- Exhausted limits (0 remaining) are highlighted in bold red
- Categories are sorted by depletion level (most depleted first)

## Flags

| Flag | Description |
|------|-------------|
| `--anonymous` | Show unauthenticated rate limits (60/hour for core) instead of authenticated limits |
| `--absolute` | Display reset times as absolute timestamps (e.g., "resets at 3:04:05 PM") |
| `--json` | Output the raw GitHub API response as JSON |

## Rate Limit Differences

The rate limits you see depend on your authentication method:

| Auth Type | Core Limit |
|-----------|------------|
| Unauthenticated | 60/hour |
| Personal Access Token | 5,000/hour |
| OAuth App | 5,000/hour |
| GitHub App (user) | 5,000-15,000/hour |
| GitHub Enterprise | 15,000/hour |

Use `gh auth status` to see your current authentication method.

## Building from Source

```bash
git clone https://github.com/sethrylan/gh-rate-limit.git
cd gh-rate-limit
go build
gh extension install .
```
