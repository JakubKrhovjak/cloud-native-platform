# Log Agent — Design Spec

**Date:** 2026-04-28
**Status:** Approved (pending implementation plan)
**Owner:** jakubkrhovjak

## 1. Goal

Build a local Python tool that uses the Anthropic SDK (Claude Sonnet 4.6) as an
agentic log monitor for the local Kind cluster `grud-cluster`. The tool runs in
two modes:

- **Periodic** — scheduled by macOS `launchd` every 15 minutes; scans logs in
  the `apps` and `infra` namespaces, decides whether anything is worth
  alerting, and posts macOS notifications for new problems.
- **On-demand** — invoked by the user as a one-shot CLI question or as an
  interactive REPL chat to investigate a current concern.

The agent is a Claude tool-use loop. Claude itself drives investigation
(which logs to fetch, which pods to describe, when to alert). Code provides
deterministic primitives (kubectl access, dedup state, macOS notifications).

## 2. Non-goals

- Not a hosted service; runs only on the user's Mac.
- Not real-time/streaming. 15-minute latency is acceptable.
- No Slack/email/webhook outputs; macOS notifications only.
- No deploy into the cluster; uses the user's local kubectl context.
- No LLM-based eval tests; agent quality is verified manually.
- Not multi-cluster aware; targets `kind-grud-cluster` context only.

## 3. Repo placement

New top-level directory `tools/log-agent/`:

```
tools/log-agent/
├── pyproject.toml          # uv/pip project, deps: anthropic, kubernetes, pyyaml
├── README.md
├── config.yaml             # namespaces, schedule, model, alert thresholds
├── log_agent/
│   ├── __init__.py
│   ├── cli.py              # entrypoint: check | ask | chat
│   ├── agent.py            # Claude tool-use loop
│   ├── tools.py            # tool definitions + handlers
│   ├── prefilter.py        # regex prefilter (ERROR|panic|OOM|FATAL|HTTP 5xx)
│   ├── state.py            # JSON state file with alert hashes + TTL
│   └── notify.py           # macOS notification via osascript
├── state/
│   └── alerts.json         # runtime state (gitignored)
├── tests/
│   ├── test_prefilter.py
│   ├── test_state.py
│   ├── test_notify.py
│   ├── test_tools_integration.py   # skipped if cluster unavailable
│   └── test_agent_loop.py          # mocks anthropic SDK
└── launchd/
    └── com.cloudnative.logagent.plist
```

The directory sits alongside `services/` rather than inside it because the
agent is a local developer tool, not a microservice deployed into the cluster.

## 4. Tool surface (Claude tools)

The agent exposes 6 tools. All read-only against the cluster except
`send_alert`, which writes to state and triggers a macOS notification.

| Tool                              | Purpose                                                                                                             |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `list_pods(namespace)`            | List pods in a namespace with phase + container statuses.                                                           |
| `get_logs(namespace, pod, since_seconds, tail_lines, grep?)` | Fetch container logs. Defaults: `tail_lines=200`, capped at 2000 (truncates with warning in response). |
| `get_pod_events(namespace, pod)`  | `kubectl describe`-style events for a pod (CrashLoopBackOff, OOMKilled, ImagePullBackOff causes).                   |
| `prefiltered_suspects(since_seconds)` | Server-side regex pass over recent logs in configured namespaces, returns `[{namespace, pod, matched_lines}]`.   |
| `was_alerted_recently(fingerprint)` | Boolean: did this fingerprint alert within `dedup_ttl_seconds`?                                                   |
| `send_alert(title, summary, fingerprint, severity)` | Marks fingerprint as alerted in state. Posts macOS notification only if `was_alerted_recently(fingerprint)` would now be false (i.e. first alert within current TTL window). `severity ∈ {info, warn, crit}` is recorded in state and used as the notification's subtitle prefix (`[CRIT]`, `[WARN]`, `[INFO]`); `crit` also plays the default macOS sound. |

### Implementation notes

- Kubernetes access via the official `kubernetes` Python client, using the
  user's existing kubeconfig and `kind-grud-cluster` context.
- `prefiltered_suspects` iterates pods in configured namespaces and runs the
  configured regex against the recent log tail. Returns only matching
  pods/lines; agent decides what to investigate further.
- `send_alert` calls `osascript -e 'display notification "<summary>" with title "<title>"'`.
- Fingerprint is the SHA-256 of `namespace + pod + error_signature` — Claude
  computes the signature (stable substring of the error). Code does not
  attempt to canonicalize.

## 5. Agent loop & entrypoints

Three entrypoints exposed via `python -m log_agent`:

```bash
# 1. Periodic check (launchd target)
python -m log_agent check
# Non-interactive. System prompt instructs Claude to: call
# prefiltered_suspects, judge each suspect (pulling more context as needed),
# check was_alerted_recently, send_alert if novel. Exit 0 on success.

# 2. One-shot question
python -m log_agent ask "what happened in project-service in the last 30 min"
# Single turn from user, agent loops until end_turn, prints final text, exits.

# 3. Interactive REPL
python -m log_agent chat
# Multi-turn conversation, history kept in memory. Ctrl+D / Ctrl+C exits.
```

### Loop pseudocode

```python
messages = [...]
iterations = 0
while True:
    resp = client.messages.create(
        model=config.model,
        system=SYSTEM_PROMPT,             # cached
        tools=TOOLS,                      # cached
        messages=messages,
        max_tokens=4096,
    )
    if resp.stop_reason == "end_turn":
        return final_text(resp)
    if resp.stop_reason == "tool_use":
        for tu in resp.content_tool_uses:
            result = handle_tool(tu.name, tu.input)
            messages.append(tool_result(tu.id, result))
        iterations += 1
        cumulative_input_tokens += resp.usage.input_tokens
        if iterations >= config.max_iterations:
            abort_with_log("max_iterations")
        if cumulative_input_tokens >= config.max_session_input_tokens:
            abort_with_log("max_session_input_tokens")
```

### Prompt caching

System prompt and tool definitions use `cache_control` so the periodic runs
within a 5-minute window benefit from cache hits. REPL also benefits across
turns.

## 6. State, config, scheduling

### State file: `state/alerts.json`

```json
{
  "alerts": {
    "<sha256>": {
      "first_seen": "2026-04-28T10:15:00Z",
      "last_alerted": "2026-04-28T10:15:00Z",
      "count": 3,
      "title": "project-service CrashLoopBackOff"
    }
  }
}
```

- TTL: 6 hours. Entries older than `last_alerted + ttl` are evicted on every
  read. If a problem persists past TTL, it re-alerts (acts as a reminder).
- `count` increments every time `send_alert` is called for the same
  fingerprint within TTL (notification is suppressed but count goes up so we
  can see at-a-glance "this fired N times").
- File lock (`fcntl.flock`) on read-modify-write so periodic and on-demand
  runs do not race.
- Corrupt JSON → renamed to `alerts.json.corrupt-<ts>`, fresh empty state
  used. Logged to stderr.

### Config: `config.yaml`

```yaml
namespaces: [apps, infra]
model: claude-sonnet-4-6
periodic:
  lookback_seconds: 900       # 15 min
  max_iterations: 25
alerts:
  dedup_ttl_seconds: 21600    # 6h
  prefilter_pattern: "ERROR|panic|OOM|FATAL|HTTP 5\\d{2}"
limits:
  max_log_lines: 2000           # per get_logs call (truncates with warning)
  max_session_input_tokens: 50000  # cumulative input tokens; loop aborts if exceeded
```

### Periodic scheduling: `launchd`

- Plist at `tools/log-agent/launchd/com.cloudnative.logagent.plist`.
- `StartInterval`: 900 seconds.
- `StandardOutPath` / `StandardErrorPath` → `~/Library/Logs/log-agent.log`.
- Installation is manual (one-time): user copies the plist to
  `~/Library/LaunchAgents/` and runs `launchctl load`. README documents
  this; the tool does not self-install.

### Auth

`ANTHROPIC_API_KEY` read from environment. CLI verifies presence at startup
and exits with a clear error message before any API call if missing.

## 7. Testing

`pytest` + `pytest-mock`.

| Layer                | Tests                                                                                       |
| -------------------- | ------------------------------------------------------------------------------------------- |
| `prefilter.py`       | Unit. Regex matches expected lines, ignores noise.                                          |
| `state.py`           | Unit. Hash stability, TTL eviction, file-lock contention, corrupt-file recovery.            |
| `notify.py`          | Unit. Mocks `subprocess.run`, asserts osascript args.                                       |
| `tools.py`           | Integration against real Kind cluster. `pytest.skip` if `kubectl get nodes` fails.          |
| `agent.py`           | Unit. Mocks `anthropic.Anthropic`; verifies tool dispatch, max-iterations abort, end_turn handling. |
| LLM evals            | Out of scope. Agent quality verified manually via `ask` mode.                               |

## 8. Error handling

| Scenario                                        | Behavior                                                                              |
| ----------------------------------------------- | ------------------------------------------------------------------------------------- |
| `ANTHROPIC_API_KEY` missing                     | Exit 1 with clear message before first API call.                                      |
| kubectl context not Kind / cluster unreachable  | Exit 2 with hint: `kubectl config use-context kind-grud-cluster`.                     |
| Anthropic API rate limit / 5xx                  | Exponential backoff retry (3 attempts), then abort the run (periodic) or surface error (interactive). |
| Tool handler raises                             | Returns to Claude as `is_error=true` tool result; Claude decides whether to retry / give up. |
| Max iterations exceeded                         | Periodic: log warning, exit 0. Interactive: print message to user.                    |
| Max session input tokens exceeded               | Same as max iterations: log warning + abort current session.                          |
| State file corrupt JSON                         | Backup `alerts.json.corrupt-<ts>`, start with empty state.                            |
| `osascript` notification fails                  | Logged to stderr; alert remains in state (not double-sent); run does not crash.       |

## 9. Out of scope (explicitly deferred)

- In-cluster deployment.
- Streaming/real-time mode.
- Multi-cluster support.
- Other notification channels (Slack, email, webhook).
- Self-installation of the launchd unit.
- Auto-remediation actions (this agent only observes).

## 10. Open follow-ups (post-MVP)

None gating the first implementation. Possible later additions: configurable
per-pod alert routing, summary digest mode (top N issues over 24h), metrics
export to Prometheus.
