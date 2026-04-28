# log-agent

Anthropic-SDK agent that monitors Kubernetes logs in the local Kind cluster `grud-cluster`.

Two modes:
- **periodic** (`check`) — invoked by launchd every 15 minutes; sends macOS notifications for new problems
- **on-demand** (`ask`, `chat`) — one-shot question or REPL

## Install

```bash
cd tools/log-agent
python3 -m venv .venv
source .venv/bin/activate
pip install -e .
```

## Configure

- `config.yaml` — namespaces, model, intervals, prefilter regex
- `ANTHROPIC_API_KEY` — required env var
- `LOG_AGENT_CONFIG` — optional path override (defaults to `tools/log-agent/config.yaml`)
- `LOG_AGENT_STATE_DIR` — optional state directory override (defaults to `tools/log-agent/state`)

## Run

```bash
# periodic check (one-shot)
python -m log_agent check

# one-shot question
python -m log_agent ask "what happened in project-service in the last 30 minutes"

# REPL
python -m log_agent chat
```

## Schedule via launchd

```bash
# 1. Edit launchd/com.cloudnative.logagent.plist — replace all REPLACE_* placeholders
# 2. Copy to LaunchAgents
cp launchd/com.cloudnative.logagent.plist ~/Library/LaunchAgents/
# 3. Load
launchctl load ~/Library/LaunchAgents/com.cloudnative.logagent.plist
# 4. Verify
launchctl list | grep cloudnative.logagent
# 5. Logs
tail -f ~/Library/Logs/log-agent.log
```

To unload:
```bash
launchctl unload ~/Library/LaunchAgents/com.cloudnative.logagent.plist
```

## Tests

```bash
pip install -e .[dev]
pytest                                              # unit tests only
pytest tests/test_tools_integration.py              # integration (needs running kind cluster)
```

## Architecture

See `docs/superpowers/specs/2026-04-28-log-agent-design.md` and `docs/superpowers/plans/2026-04-28-log-agent.md`.
