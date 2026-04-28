import shutil
import subprocess
from pathlib import Path

import pytest

from kubernetes import client as k8s_client, config as k8s_config

from log_agent.config import Config, PeriodicConfig, AlertsConfig, LimitsConfig
from log_agent.state import AlertState
from log_agent.tools import ToolContext, handle_list_pods, handle_prefiltered_suspects


def _kind_available() -> bool:
    if shutil.which("kubectl") is None:
        return False
    try:
        result = subprocess.run(
            ["kubectl", "--context", "kind-grud-cluster", "get", "nodes"],
            capture_output=True, timeout=5,
        )
        return result.returncode == 0
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return False


pytestmark = pytest.mark.skipif(
    not _kind_available(),
    reason="kind-grud-cluster context not available",
)


@pytest.fixture
def real_ctx(tmp_path):
    k8s_config.load_kube_config(context="kind-grud-cluster")
    cfg = Config(
        namespaces=["apps", "infra"],
        model="claude-sonnet-4-6",
        periodic=PeriodicConfig(lookback_seconds=600, max_iterations=5),
        alerts=AlertsConfig(dedup_ttl_seconds=3600, prefilter_pattern=r"ERROR|panic|OOM|FATAL"),
        limits=LimitsConfig(max_log_lines=200, max_session_input_tokens=50000),
    )
    return ToolContext(
        config=cfg,
        state=AlertState(tmp_path / "alerts.json", ttl_seconds=3600),
        core_v1=k8s_client.CoreV1Api(),
    )


def test_list_pods_apps_namespace_returns_real_pods(real_ctx):
    result = handle_list_pods(real_ctx, {"namespace": "apps"})
    assert "pods" in result
    pod_names = [p["name"] for p in result["pods"]]
    assert any("student-service" in n for n in pod_names), pod_names
    assert any("project-service" in n for n in pod_names), pod_names


def test_prefiltered_suspects_runs_without_error(real_ctx):
    result = handle_prefiltered_suspects(real_ctx, {"since_seconds": 600})
    assert "suspects" in result
    # Don't assert on content — cluster may be healthy
