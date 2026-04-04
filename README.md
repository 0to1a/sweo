# sweo

SWE-Orchestrator — spawns and manages parallel AI coding agents (Claude Code, Codex) across GitHub issues. Assigns issues to agents, creates worktrees, tracks progress via a web dashboard, and handles the full lifecycle from issue pickup to PR creation.

## Prerequisites

- Go >= 1.25 ([install guide](https://go.dev/doc/install))
- [tmux](https://github.com/tmux/tmux) — used to run agent sessions
- [gh CLI](https://cli.github.com/) — GitHub interaction (must be authenticated)
- [Git](https://git-scm.com/)
- An AI coding agent:
  - [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`claude` binary), or
  - [Codex](https://github.com/openai/codex) (`codex` binary)
- (Optional) [Bun](https://bun.sh/) — only needed if building the frontend from source

## Installation

```bash
git clone https://github.com/0to1a/sweo.git
cd sweo
go mod download
make build
```

This produces a `sweo` binary in the project root.

To build with the frontend dashboard included:

```bash
make build-all
```

### Cross-compile

```bash
make build-linux    # linux/amd64
make build-darwin   # darwin/arm64
```

## Configuration

On first run, sweo creates a default config at `~/.sweo/config.yaml`. Edit it to add your project(s).

```yaml
port: 3000
delayCheckIssue: 30
delayCheckChanges: 30
maximumInactiveMins: 10
webhookUrl: ""

projects:
  my-project:
    repo: "org/repo"
    path: "/home/you/code/repo"
    defaultBranch: "main"
    agent: "claude-code"
    waitingCI: true
```

| Variable | Description | Default | Required |
|---|---|---|---|
| `port` | Web dashboard port | `3000` | no |
| `delayCheckIssue` | Seconds between issue polling cycles | `30` | no |
| `delayCheckChanges` | Seconds between PR/CI/review polling cycles | `30` | no |
| `maximumInactiveMins` | Minutes of inactivity before marking a session as stuck | `10` | no |
| `webhookUrl` | Outbound webhook URL (empty = disabled) | `""` | no |

### Project settings

| Field | Description | Default | Required |
|---|---|---|---|
| `repo` | GitHub `org/repo` | — | yes |
| `path` | Local checkout path | — | yes |
| `defaultBranch` | Base branch for worktrees | `main` | no |
| `agent` | AI agent to use (`claude-code` or `codex`) | `claude-code` | no |
| `agentRules` | Custom instructions passed to the agent | `""` | no |
| `waitingCI` | Wait for CI to pass before advancing state | `true` | no |

## Run

```bash
./sweo start
```

This starts the orchestrator and the web dashboard.

## Verify

```bash
curl http://localhost:3000/health
```

You should see the dashboard at `http://localhost:3000`.

## Commands

| Command | Description |
|---|---|
| `sweo start` | Start the orchestrator and web dashboard |
| `sweo status` | Show current session status |
| `sweo doctor` | Check system dependencies and configuration |
| `sweo version` | Print version |

## Development

Run the Go server in dev mode:

```bash
make dev
```

Run tests:

```bash
make test
```
