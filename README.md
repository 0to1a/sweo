# sweo

SWE-Orchestrator — spawns and manages parallel AI coding agents (Claude Code, Codex) across GitHub issues using tmux sessions and git worktrees.

## Prerequisites

- Go >= 1.25 ([install guide](https://go.dev/doc/install))
- [tmux](https://github.com/tmux/tmux) — session management for agents
- [gh CLI](https://cli.github.com/) — GitHub integration (must be authenticated via `gh auth login`)
- [Bun](https://bun.sh/) — frontend build toolchain
- An AI coding agent:
  - [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`claude` binary), or
  - [Codex](https://github.com/openai/codex) (`codex` binary)

## Installation

```bash
git clone https://github.com/0to1a/sweo.git
cd sweo
go mod download
```

Build the binary (Go backend only):

```bash
make build
```

Build everything (frontend + backend):

```bash
make build-all
```

## Configuration

On first run, sweo creates a default config at `~/.sweo/config.yaml`. Edit it to add your project(s).

| Variable | Description | Default | Required |
|---|---|---|---|
| `port` | Web dashboard port | `3000` | no |
| `delayCheckIssue` | Seconds between issue polling cycles | `30` | no |
| `delayCheckChanges` | Seconds between PR/CI/review polling cycles | `30` | no |
| `maximumInactiveMins` | Minutes of inactivity before marking session as stuck | `10` | no |
| `webhookUrl` | Outbound webhook URL (empty = disabled) | `""` | no |

### Project configuration

Each project entry under `projects:` supports:

| Field | Description | Default | Required |
|---|---|---|---|
| `repo` | GitHub `org/repo` | — | yes |
| `path` | Local checkout path | — | yes |
| `defaultBranch` | Base branch for worktrees | `main` | no |
| `agent` | AI agent to use (`claude-code` or `codex`) | `claude-code` | no |
| `agentRules` | Custom instructions passed to the agent | `""` | no |
| `waitingCI` | Wait for CI before advancing session state | `true` | no |

Example `~/.sweo/config.yaml`:

```yaml
port: 3000

projects:
  my-project:
    repo: "org/repo"
    path: "/home/you/code/repo"
    defaultBranch: "main"
    agent: "claude-code"
    agentRules: |
      Always write tests.
    waitingCI: true
```

## Run

Start the orchestrator and web dashboard:

```bash
./sweo start
```

Or run directly without building:

```bash
go run ./cmd/sweo/ start
```

Other commands:

```bash
sweo doctor   # Check system dependencies and configuration
sweo status   # Show current session status
sweo version  # Print version
```

## Verify

1. Check that all dependencies are installed:

```bash
./sweo doctor
```

2. Start the orchestrator:

```bash
./sweo start
```

Expected output:

```
sweo — SWE-Orchestrator

  Dashboard:  http://localhost:3000
  Projects:   1
    • my-project (org/repo, agent: claude-code)

Press Ctrl+C to stop
```

3. Open the dashboard:

```bash
curl http://localhost:3000
```

## Development

Run the Go server in dev mode:

```bash
make dev
```

Run tests:

```bash
make test
```
