import argparse
import logging
import os
import sys
from pathlib import Path
from collections.abc import Sequence

from anthropic import Anthropic
from kubernetes import client as k8s_client, config as k8s_config

from log_agent.agent import Agent
from log_agent.config import load_config
from log_agent.state import AlertState
from log_agent.tools import ToolContext

log = logging.getLogger(__name__)


SYSTEM_PROMPT_PERIODIC = """You are a Kubernetes log monitor for the local Kind cluster `grud-cluster`.

You run every ~15 minutes. Your job:
1. Call `prefiltered_suspects` with the configured lookback window to find pods with suspicious log lines.
2. For each suspect, decide if it is a real problem worth a notification (transient retries, debug noise, expected errors → ignore).
3. Use `get_pod_events` and `get_logs` (with `grep` to focus) to gather supporting evidence when needed.
4. For each real problem: pick a stable error_signature (a minimal substring of the error that doesn't contain timestamps or request IDs). Call `was_alerted_recently(namespace, pod, error_signature)` first; only call `send_alert(...)` with the same components if was_alerted_recently returned false.
5. End your turn with a brief one-line summary (e.g. "1 new alert sent, 2 suspects ignored").

Be terse. Do not over-investigate. If nothing suspicious, end immediately."""

SYSTEM_PROMPT_INTERACTIVE = """You are a Kubernetes log assistant for the local Kind cluster `grud-cluster`.

The user asks you questions about cluster health. Use the tools to fetch facts; do not guess.
Available namespaces are configured. Reply in the user's language. Be concrete and concise."""


def _build_agent(*, system_prompt: str) -> Agent:
    cfg_path = Path(os.environ.get("LOG_AGENT_CONFIG", Path(__file__).parent.parent / "config.yaml"))
    if not cfg_path.exists():
        raise RuntimeError(
            f"config not found at {cfg_path}. "
            f"Set LOG_AGENT_CONFIG to point to your config.yaml."
        )
    cfg = load_config(cfg_path)

    state_dir = Path(os.environ.get("LOG_AGENT_STATE_DIR", Path(__file__).parent.parent / "state"))
    state = AlertState(state_dir / "alerts.json", ttl_seconds=cfg.alerts.dedup_ttl_seconds)

    try:
        k8s_config.load_kube_config()
    except Exception as e:
        raise RuntimeError(
            f"failed to load kubeconfig: {e}\n"
            f"hint: kubectl config use-context kind-grud-cluster"
        ) from e

    core_v1 = k8s_client.CoreV1Api()
    ctx = ToolContext(config=cfg, state=state, core_v1=core_v1)
    client = Anthropic()
    return Agent(ctx=ctx, client=client, system_prompt=system_prompt)


def _check_env() -> None:
    if not os.environ.get("ANTHROPIC_API_KEY"):
        print("error: ANTHROPIC_API_KEY is not set", file=sys.stderr)
        raise SystemExit(1)


def _cmd_check(args) -> int:
    agent = _build_agent(system_prompt=SYSTEM_PROMPT_PERIODIC)
    user_msg = (
        f"Run periodic check now. Lookback: "
        f"{agent.ctx.config.periodic.lookback_seconds} seconds."
    )
    result = agent.run(user_msg)
    print(result.text)
    if result.aborted:
        log.warning("check aborted: %s", result.abort_reason)
    return 0


def _cmd_ask(args) -> int:
    agent = _build_agent(system_prompt=SYSTEM_PROMPT_INTERACTIVE)
    result = agent.run(args.question)
    print(result.text)
    return 0


def _cmd_chat(args) -> int:
    agent = _build_agent(system_prompt=SYSTEM_PROMPT_INTERACTIVE)
    print("log-agent chat (Ctrl+D to exit)")
    while True:
        try:
            line = input("you> ").strip()
        except (EOFError, KeyboardInterrupt):
            print()
            return 0
        if not line:
            continue
        try:
            result = agent.run(line)
        except Exception as e:
            print(f"agent> error: {type(e).__name__}: {e}", file=sys.stderr)
            continue
        print(f"agent> {result.text}", flush=True)


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="log-agent")
    sub = parser.add_subparsers(dest="cmd", required=True)

    sub.add_parser("check", help="Run one periodic check (intended for launchd).")

    p_ask = sub.add_parser("ask", help="One-shot question.")
    p_ask.add_argument("question", help="Question for the agent.")

    sub.add_parser("chat", help="Interactive REPL chat.")

    args = parser.parse_args(argv)
    _check_env()

    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")

    try:
        if args.cmd == "check":
            return _cmd_check(args)
        if args.cmd == "ask":
            return _cmd_ask(args)
        if args.cmd == "chat":
            return _cmd_chat(args)
    except RuntimeError as e:
        print(f"error: {e}", file=sys.stderr)
        return 2
    return 0  # unreachable; satisfies type
