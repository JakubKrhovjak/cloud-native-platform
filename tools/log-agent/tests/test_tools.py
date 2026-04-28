from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest

from log_agent.tools import (
    ToolContext,
    handle_list_pods,
    handle_get_logs,
    handle_get_pod_events,
    handle_prefiltered_suspects,
    handle_was_alerted_recently,
    handle_send_alert,
)


@pytest.fixture
def ctx(tmp_path, mocker):
    from log_agent.config import Config, PeriodicConfig, AlertsConfig, LimitsConfig
    from log_agent.state import AlertState

    cfg = Config(
        namespaces=["apps", "infra"],
        model="claude-sonnet-4-6",
        periodic=PeriodicConfig(lookback_seconds=900, max_iterations=25),
        alerts=AlertsConfig(dedup_ttl_seconds=3600, prefilter_pattern=r"ERROR|panic"),
        limits=LimitsConfig(max_log_lines=2000, max_session_input_tokens=50000),
    )
    state = AlertState(tmp_path / "alerts.json", ttl_seconds=3600)
    core_v1 = MagicMock()
    notify = mocker.patch("log_agent.tools.notify_macos", return_value=True)
    return ToolContext(config=cfg, state=state, core_v1=core_v1, notify=notify)


def test_list_pods_returns_pod_summaries(ctx):
    ctx.core_v1.list_namespaced_pod.return_value = SimpleNamespace(items=[
        SimpleNamespace(
            metadata=SimpleNamespace(name="student-service-abc", namespace="apps"),
            status=SimpleNamespace(
                phase="Running",
                container_statuses=[
                    SimpleNamespace(name="app", ready=True, restart_count=0, state=SimpleNamespace(running=True, waiting=None, terminated=None)),
                ],
            ),
        )
    ])
    result = handle_list_pods(ctx, {"namespace": "apps"})
    assert result["namespace"] == "apps"
    assert len(result["pods"]) == 1
    assert result["pods"][0]["name"] == "student-service-abc"
    assert result["pods"][0]["phase"] == "Running"
    assert result["pods"][0]["containers"][0]["ready"] is True


def test_list_pods_rejects_unknown_namespace(ctx):
    result = handle_list_pods(ctx, {"namespace": "kube-system"})
    assert "error" in result
    assert "not in configured namespaces" in result["error"]


def test_get_logs_caps_tail_lines(ctx):
    ctx.core_v1.read_namespaced_pod_log.return_value = "line1\nline2\nline3"
    result = handle_get_logs(ctx, {
        "namespace": "apps",
        "pod": "x",
        "since_seconds": 600,
        "tail_lines": 10000,
    })
    call_kwargs = ctx.core_v1.read_namespaced_pod_log.call_args.kwargs
    assert call_kwargs["tail_lines"] == 2000  # capped to limits.max_log_lines
    assert call_kwargs["since_seconds"] == 600
    assert result["truncated"] is True
    assert result["lines"] == ["line1", "line2", "line3"]


def test_get_logs_default_tail(ctx):
    ctx.core_v1.read_namespaced_pod_log.return_value = "x"
    handle_get_logs(ctx, {"namespace": "apps", "pod": "p", "since_seconds": 60})
    assert ctx.core_v1.read_namespaced_pod_log.call_args.kwargs["tail_lines"] == 200


def test_get_logs_with_grep_filters(ctx):
    ctx.core_v1.read_namespaced_pod_log.return_value = "INFO ok\nERROR bad\nINFO done"
    result = handle_get_logs(ctx, {
        "namespace": "apps",
        "pod": "p",
        "since_seconds": 60,
        "grep": "ERROR",
    })
    assert result["lines"] == ["ERROR bad"]


def test_get_pod_events(ctx):
    ctx.core_v1.list_namespaced_event.return_value = SimpleNamespace(items=[
        SimpleNamespace(
            type="Warning",
            reason="BackOff",
            message="Back-off restarting failed container",
            last_timestamp=None,
            involved_object=SimpleNamespace(name="p", kind="Pod"),
        )
    ])
    result = handle_get_pod_events(ctx, {"namespace": "apps", "pod": "p"})
    assert len(result["events"]) == 1
    assert result["events"][0]["reason"] == "BackOff"


def test_was_alerted_recently_false_initially(ctx):
    result = handle_was_alerted_recently(ctx, {
        "namespace": "apps", "pod": "p", "error_signature": "panic"
    })
    assert result["was_alerted"] is False
    assert "fingerprint" in result


def test_send_alert_records_and_notifies(ctx):
    result = handle_send_alert(ctx, {
        "title": "t",
        "summary": "s",
        "namespace": "apps",
        "pod": "p",
        "error_signature": "panic",
        "severity": "warn",
    })
    assert result["notified"] is True
    assert result["was_new"] is True
    ctx.notify.assert_called_once_with(
        title="t",
        subtitle="[WARN]",
        message="s",
        play_sound=False,
    )


def test_send_alert_dedups_within_ttl(ctx):
    common = {"title": "t", "summary": "s", "namespace": "apps", "pod": "p", "error_signature": "panic", "severity": "warn"}
    handle_send_alert(ctx, common)
    ctx.notify.reset_mock()
    result = handle_send_alert(ctx, common)
    assert result["notified"] is False
    assert result["was_new"] is False
    ctx.notify.assert_not_called()


def test_send_alert_crit_plays_sound(ctx):
    handle_send_alert(ctx, {"title": "t", "summary": "s", "namespace": "apps", "pod": "p", "error_signature": "panic", "severity": "crit"})
    assert ctx.notify.call_args.kwargs["play_sound"] is True


def test_send_alert_fingerprint_is_stable_sha256(ctx):
    from hashlib import sha256
    args = {"title": "t", "summary": "s", "namespace": "apps", "pod": "p", "error_signature": "panic: nil"}
    result = handle_send_alert(ctx, {**args, "severity": "warn"})
    expected = sha256(b"apps|p|panic: nil").hexdigest()
    assert result["fingerprint"] == expected


def test_prefiltered_suspects_returns_matches(ctx):
    ctx.core_v1.list_namespaced_pod.side_effect = [
        SimpleNamespace(items=[SimpleNamespace(
            metadata=SimpleNamespace(name="p1", namespace="apps"),
            status=SimpleNamespace(phase="Running", container_statuses=[]),
        )]),
        SimpleNamespace(items=[]),  # infra ns empty
    ]
    ctx.core_v1.read_namespaced_pod_log.return_value = "INFO ok\nERROR bad\nINFO done"
    result = handle_prefiltered_suspects(ctx, {"since_seconds": 600})
    assert len(result["suspects"]) == 1
    assert result["suspects"][0]["namespace"] == "apps"
    assert result["suspects"][0]["pod"] == "p1"
    assert any("ERROR bad" in line for line in result["suspects"][0]["matched_lines"])


def test_get_logs_truncated_when_log_fills_tail(ctx):
    """truncated must be True when returned lines >= requested tail."""
    ctx.core_v1.read_namespaced_pod_log.return_value = "\n".join(f"line{i}" for i in range(200))
    result = handle_get_logs(ctx, {"namespace": "apps", "pod": "p", "since_seconds": 60})
    # default tail_lines=200, returned 200 lines -> cap was likely hit
    assert result["truncated"] is True


def test_get_logs_not_truncated_when_under_tail(ctx):
    ctx.core_v1.read_namespaced_pod_log.return_value = "line1\nline2"
    result = handle_get_logs(ctx, {"namespace": "apps", "pod": "p", "since_seconds": 60})
    assert result["truncated"] is False


def test_get_pod_events_serializes_real_timestamp(ctx):
    from datetime import datetime, timezone
    ts = datetime(2026, 4, 28, 10, 0, 0, tzinfo=timezone.utc)
    ctx.core_v1.list_namespaced_event.return_value = SimpleNamespace(items=[
        SimpleNamespace(
            type="Warning",
            reason="OOMKilled",
            message="container killed",
            last_timestamp=ts,
            involved_object=SimpleNamespace(name="p", kind="Pod"),
        )
    ])
    result = handle_get_pod_events(ctx, {"namespace": "apps", "pod": "p"})
    assert result["events"][0]["last_timestamp"] == "2026-04-28T10:00:00+00:00"
