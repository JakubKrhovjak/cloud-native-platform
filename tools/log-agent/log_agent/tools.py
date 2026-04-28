from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from log_agent.config import Config
from log_agent.notify import notify_macos
from log_agent.prefilter import compile_pattern, match_lines
from log_agent.state import AlertState, fingerprint


@dataclass
class ToolContext:
    config: Config
    state: AlertState
    core_v1: Any  # kubernetes.client.CoreV1Api
    notify: Callable[..., bool] = notify_macos


# ---------- Anthropic tool schemas ----------

TOOL_SCHEMAS: list[dict[str, Any]] = [
    {
        "name": "list_pods",
        "description": "List pods in a configured namespace with phase and container statuses.",
        "input_schema": {
            "type": "object",
            "properties": {
                "namespace": {"type": "string", "description": "Namespace (must be in configured namespaces)"},
            },
            "required": ["namespace"],
        },
    },
    {
        "name": "get_logs",
        "description": "Fetch recent container logs for a pod. tail_lines is capped at the configured limit.",
        "input_schema": {
            "type": "object",
            "properties": {
                "namespace": {"type": "string", "description": "Namespace (must be in configured namespaces)"},
                "pod": {"type": "string", "description": "Pod name"},
                "since_seconds": {
                    "type": "integer",
                    "description": (
                        "Lookback window in seconds. Typically use the configured "
                        "lookback_seconds from the system prompt (e.g. 900 for 15 minutes). "
                        "Larger values may hit per-pod log caps."
                    ),
                },
                "tail_lines": {"type": "integer", "description": "Max lines to return (default 200, capped)"},
                "grep": {"type": "string", "description": "Optional regex to filter lines"},
            },
            "required": ["namespace", "pod", "since_seconds"],
        },
    },
    {
        "name": "get_pod_events",
        "description": "Fetch Kubernetes events for a pod (kubectl describe-style: BackOff, OOMKilled, ImagePullBackOff, etc).",
        "input_schema": {
            "type": "object",
            "properties": {
                "namespace": {"type": "string", "description": "Namespace (must be in configured namespaces)"},
                "pod": {"type": "string", "description": "Pod name"},
            },
            "required": ["namespace", "pod"],
        },
    },
    {
        "name": "prefiltered_suspects",
        "description": "Run a regex prefilter across recent logs in all configured namespaces and return suspect pods + matched lines (capped at 50 per pod; matched_lines_truncated indicates if more existed).",
        "input_schema": {
            "type": "object",
            "properties": {
                "since_seconds": {
                    "type": "integer",
                    "description": (
                        "Lookback window in seconds. Typically use the configured "
                        "lookback_seconds from the system prompt (e.g. 900 for 15 minutes). "
                        "Larger values may hit per-pod log caps."
                    ),
                },
            },
            "required": ["since_seconds"],
        },
    },
    {
        "name": "was_alerted_recently",
        "description": "Check whether an alert with these semantic components has been alerted within the dedup TTL. The handler computes a stable SHA-256 fingerprint from the inputs.",
        "input_schema": {
            "type": "object",
            "properties": {
                "namespace": {"type": "string", "description": "Kubernetes namespace of the affected pod"},
                "pod": {"type": "string", "description": "Pod name (full, including suffix)"},
                "error_signature": {
                    "type": "string",
                    "description": (
                        "Stable, minimal substring of the error (e.g. exception class name + first line of message). "
                        "Must NOT contain timestamps, request IDs, or memory addresses, since these vary across recurrences. "
                        "The same recurring problem must produce the same error_signature across runs."
                    ),
                },
            },
            "required": ["namespace", "pod", "error_signature"],
        },
    },
    {
        "name": "send_alert",
        "description": "Record an alert in dedup state and post a macOS notification if this is the first occurrence within TTL. Fingerprint is computed server-side from the semantic components.",
        "input_schema": {
            "type": "object",
            "properties": {
                "title": {"type": "string", "description": "Short alert title (max ~80 chars), shown as macOS notification title"},
                "summary": {"type": "string", "description": "One-paragraph summary of the problem with key context (max ~500 chars)"},
                "namespace": {"type": "string", "description": "Kubernetes namespace of the affected pod"},
                "pod": {"type": "string", "description": "Pod name (full)"},
                "error_signature": {
                    "type": "string",
                    "description": (
                        "Stable, minimal substring of the error (e.g. exception class name + first line of message). "
                        "Must NOT contain timestamps, request IDs, or memory addresses. "
                        "The same recurring problem must produce the same error_signature."
                    ),
                },
                "severity": {"type": "string", "enum": ["info", "warn", "crit"], "description": "info: noted but low-impact. warn: degradation. crit: outage or data loss."},
            },
            "required": ["title", "summary", "namespace", "pod", "error_signature", "severity"],
        },
        "cache_control": {"type": "ephemeral"},
    },
]


# ---------- Handlers ----------

def _check_namespace(ctx: ToolContext, ns: str) -> dict[str, Any] | None:
    if ns not in ctx.config.namespaces:
        return {"error": f"namespace {ns!r} not in configured namespaces {ctx.config.namespaces}"}
    return None


def _container_status_dict(cs: Any) -> dict[str, Any]:
    state = "unknown"
    if cs.state.running:
        state = "running"
    elif cs.state.waiting:
        state = f"waiting:{cs.state.waiting.reason or '?'}"
    elif cs.state.terminated:
        state = f"terminated:{cs.state.terminated.reason or '?'}"
    return {
        "name": cs.name,
        "ready": cs.ready,
        "restart_count": cs.restart_count,
        "state": state,
    }


def handle_list_pods(ctx: ToolContext, args: dict[str, Any]) -> dict[str, Any]:
    ns = args["namespace"]
    err = _check_namespace(ctx, ns)
    if err:
        return err
    resp = ctx.core_v1.list_namespaced_pod(namespace=ns)
    pods = []
    for p in resp.items:
        containers = [_container_status_dict(c) for c in (p.status.container_statuses or [])]
        pods.append({
            "name": p.metadata.name,
            "phase": p.status.phase,
            "containers": containers,
        })
    return {"namespace": ns, "pods": pods}


def handle_get_logs(ctx: ToolContext, args: dict[str, Any]) -> dict[str, Any]:
    ns = args["namespace"]
    err = _check_namespace(ctx, ns)
    if err:
        return err
    requested = args.get("tail_lines", 200)
    cap = ctx.config.limits.max_log_lines
    tail = min(requested, cap)
    raw = ctx.core_v1.read_namespaced_pod_log(
        name=args["pod"],
        namespace=ns,
        since_seconds=args["since_seconds"],
        tail_lines=tail,
        timestamps=True,
    )
    raw_lines = raw.splitlines() if raw else []
    actually_capped = len(raw_lines) >= tail
    grep = args.get("grep")
    if grep:
        pat = compile_pattern(grep)
        lines = [line for line in raw_lines if pat.search(line)]
    else:
        lines = raw_lines
    return {
        "namespace": ns,
        "pod": args["pod"],
        "lines": lines,
        "truncated": (requested > cap) or actually_capped,
    }


def handle_get_pod_events(ctx: ToolContext, args: dict[str, Any]) -> dict[str, Any]:
    ns = args["namespace"]
    err = _check_namespace(ctx, ns)
    if err:
        return err
    field_selector = f"involvedObject.name={args['pod']}"
    resp = ctx.core_v1.list_namespaced_event(namespace=ns, field_selector=field_selector)
    events = []
    for e in resp.items:
        events.append({
            "type": e.type,
            "reason": e.reason,
            "message": e.message,
            "last_timestamp": e.last_timestamp.isoformat() if e.last_timestamp else None,
        })
    return {"namespace": ns, "pod": args["pod"], "events": events}


def handle_prefiltered_suspects(ctx: ToolContext, args: dict[str, Any]) -> dict[str, Any]:
    pattern = compile_pattern(ctx.config.alerts.prefilter_pattern)
    since = args["since_seconds"]
    suspects = []
    for ns in ctx.config.namespaces:
        pods_resp = ctx.core_v1.list_namespaced_pod(namespace=ns)
        for pod in pods_resp.items:
            try:
                raw = ctx.core_v1.read_namespaced_pod_log(
                    name=pod.metadata.name,
                    namespace=ns,
                    since_seconds=since,
                    tail_lines=ctx.config.limits.max_log_lines,
                    timestamps=True,
                )
            except Exception:
                continue
            lines = raw.splitlines() if raw else []
            matched = match_lines(pattern, lines)
            if matched:
                matched_lines_full = [line for _, line in matched]
                suspects.append({
                    "namespace": ns,
                    "pod": pod.metadata.name,
                    "matched_lines": matched_lines_full[:50],
                    "matched_lines_truncated": len(matched_lines_full) > 50,
                })
    return {"since_seconds": since, "suspects": suspects}


def handle_was_alerted_recently(ctx: ToolContext, args: dict[str, Any]) -> dict[str, Any]:
    fp = fingerprint(args["namespace"], args["pod"], args["error_signature"])
    return {"was_alerted": ctx.state.was_alerted_recently(fp), "fingerprint": fp}


def handle_send_alert(ctx: ToolContext, args: dict[str, Any]) -> dict[str, Any]:
    fp = fingerprint(args["namespace"], args["pod"], args["error_signature"])
    severity = args["severity"]
    is_new = ctx.state.record_alert(fp, title=args["title"], severity=severity)
    notified = False
    if is_new:
        prefix = f"[{severity.upper()}]"
        notified = ctx.notify(
            title=args["title"],
            subtitle=prefix,
            message=args["summary"],
            play_sound=(severity == "crit"),
        )
    return {"was_new": is_new, "notified": notified, "fingerprint": fp}


HANDLERS: dict[str, Callable[[ToolContext, dict[str, Any]], dict[str, Any]]] = {
    "list_pods": handle_list_pods,
    "get_logs": handle_get_logs,
    "get_pod_events": handle_get_pod_events,
    "prefiltered_suspects": handle_prefiltered_suspects,
    "was_alerted_recently": handle_was_alerted_recently,
    "send_alert": handle_send_alert,
}


def dispatch(ctx: ToolContext, name: str, args: dict[str, Any]) -> dict[str, Any]:
    handler = HANDLERS.get(name)
    if handler is None:
        return {"error": f"unknown tool {name!r}"}
    try:
        return handler(ctx, args)
    except Exception as e:
        return {"error": f"{type(e).__name__}: {e}"}
