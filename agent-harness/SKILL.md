---
name: newapi-cli
description: CLI for managing Atius NewAPI gateway (channels, models, containers)
---

# newapi-cli

CLI for managing Atius NewAPI LLM gateway.

## Installation

```bash
pip install -e agent-harness/
```

## Commands

### container

Docker container management.

```bash
newapi-cli container list              # List containers
newapi-cli container status            # Detailed status
newapi-cli container logs --follow     # Follow logs
newapi-cli container restart           # Restart container
```

### channel

Channel management for NewAPI.

```bash
newapi-cli channel list                  # List all channels
newapi-cli channel info --name deepseek  # Get channel details
newapi-cli channel create --name xxx --model xxx --base-url xxx
newapi-cli channel delete --name xxx
```

### model

Model information.

```bash
newapi-cli model list       # List available models
newapi-cli model info --name deepseek-chat  # Get model metadata
```

## Options

- `--base-url URL` — NewAPI base URL (default: http://localhost:3300)
- `--json` — JSON output for machine-readable results

## Examples

```bash
# List models in JSON format
newapi-cli model list --json

# Get channel info
newapi-cli channel info --name deepseek-chat

# Check container status
newapi-cli container status
```
