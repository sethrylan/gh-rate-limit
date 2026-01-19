# gh-rate-limit

A `gh` CLI extension that displays your current GitHub API rate limit status.

## Overview

`gh rate-limit` provides a terminal-based view of all GitHub API rate limit categories, with visual progress bars and color-coded status indicators. It integrates with the `gh` CLI authentication system and supports both authenticated and anonymous rate limit checking.

## Installation

```bash
gh extension install sethrylan/gh-rate-limit
```

## Usage

```bash
gh rate-limit [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--anonymous` | Show unauthenticated rate limits instead of authenticated |
| `--absolute` | Display reset times as absolute timestamps instead of relative |
| `--json` | Output raw GitHub API response as JSON |

## Technical Specification

### Stack

- **Language**: Go 1.23+
- **Key Dependency**: `github.com/cli/go-gh/v2` for authentication, API access, and terminal utilities
- **Repository**: `gh-rate-limit`

### API Endpoint

The extension calls the GitHub REST API:
```
GET https://api.github.com/rate_limit
```

For authenticated requests, it uses the token from `gh auth` configuration via `go-gh`.

### Authentication Behavior

| Scenario | Behavior |
|----------|----------|
| Authenticated (default) | Use `gh` CLI auth token, display authenticated rate limits |
| Authenticated + `--anonymous` | Ignore auth, display unauthenticated rate limits |
| Not authenticated (no `--anonymous`) | Error with hint: "Not authenticated. Run `gh auth login` or use `--anonymous`" |
| Not authenticated + `--anonymous` | Display unauthenticated rate limits |

### Output Format

#### Default (TTY/Interactive)

Rich terminal UI with:
- Aligned columns (tabular format)
- Unicode block progress bars (`█▓▒░`)
- Color gradient based on remaining percentage (256-color mode)
- Terminal-responsive progress bar width

#### Piped/Non-TTY

Plain text table without colors or visual embellishments.

#### JSON Mode (`--json`)

Raw GitHub API response passed through unchanged.

### Display Elements

For each rate limit category, display:

1. **Category name** - Raw API names (e.g., `core`, `graphql`, `code_scanning_upload`)
2. **Remaining/Limit** - Counts only, e.g., `4500/5000`
3. **Progress bar** - Visual representation of usage
4. **Reset time** - Relative by default (e.g., "5m 30s"), absolute with `--absolute` flag

### Sorting

Categories are sorted by **depletion level** (lowest percentage remaining first), so the most urgent limits appear at the top.

### Categories

Display **all** rate limit categories returned by the API, regardless of usage level.

### Color Scheme

Continuous gradient based on percentage remaining using 256-color terminal codes:

| Remaining % | Color Range |
|-------------|-------------|
| > 50% | Green |
| 20-50% | Yellow/Orange gradient |
| < 20% | Red |

### Exhausted Limits

When a rate limit reaches 0 remaining:
- Display with **bold red** text for visual emphasis
- Standard progress bar (empty)

### Progress Bar

- **Characters**: Unicode block elements (`█▓▒░`)
- **Width**: Responsive to terminal width
- **Filled portion**: Represents percentage used (not remaining)

### Error Handling

Match `gh` CLI error formatting style:
- Errors output to stderr
- Helpful suggestions included (e.g., "Run `gh auth login` to authenticate")

### Exit Codes

Standard exit codes:
- `0` - Success
- `1` - Error (API failure, authentication issue, etc.)

No semantic exit codes based on rate limit status.

## Example Output

### Default (Authenticated, TTY)

```
core                 4832/5000  ████████████████████░░░░  resets in 42m 15s
search                 28/30   █████████████████████░░░  resets in 58s
graphql              4990/5000  ████████████████████████  resets in 42m 15s
code_scanning_upload 1000/1000  ████████████████████████  resets in 1h 0m
```

### With `--absolute`

```
core                 4832/5000  ████████████████████░░░░  resets at 2:45:30 PM
search                 28/30   █████████████████████░░░  resets at 2:04:15 PM
...
```

### With `--json`

```json
{
  "resources": {
    "core": {
      "limit": 5000,
      "remaining": 4832,
      "reset": 1705340730,
      "used": 168
    },
    ...
  },
  "rate": { ... }
}
```

### Error (Not Authenticated)

```
Error: Not authenticated
Run `gh auth login` to authenticate, or use `gh rate-limit --anonymous` to check unauthenticated rate limits.
```

## Distribution

- Installable via `gh extension install sethrylan/gh-rate-limit`
- GitHub Releases with pre-built binaries (manual release process)

## Non-Goals (Explicitly Excluded)

- Watch mode / continuous polling
- Filtering by specific categories
- Multiple category selection
- Token/user information display
- Configurable color thresholds
- Semantic exit codes based on rate limit status
- `--hostname` flag for GitHub Enterprise (uses current `gh` auth only)
