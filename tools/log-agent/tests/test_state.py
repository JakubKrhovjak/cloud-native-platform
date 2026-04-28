import json
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path

from log_agent.state import AlertState, fingerprint


def _ts(seconds_ago: int) -> str:
    return (datetime.now(timezone.utc) - timedelta(seconds=seconds_ago)).isoformat()


def test_fingerprint_stable():
    a = fingerprint("apps", "project-service", "panic: nil pointer")
    b = fingerprint("apps", "project-service", "panic: nil pointer")
    assert a == b
    assert len(a) == 64  # sha256 hex


def test_fingerprint_distinguishes_pods():
    a = fingerprint("apps", "project-service", "x")
    b = fingerprint("apps", "student-service", "x")
    assert a != b


def test_was_alerted_recently_false_for_unknown(tmp_path: Path):
    state = AlertState(tmp_path / "alerts.json", ttl_seconds=3600)
    assert state.was_alerted_recently("deadbeef") is False


def test_record_then_was_alerted_recently_true(tmp_path: Path):
    state = AlertState(tmp_path / "alerts.json", ttl_seconds=3600)
    fp = "deadbeef"
    state.record_alert(fp, title="x", severity="warn")
    assert state.was_alerted_recently(fp) is True


def test_record_alert_increments_count(tmp_path: Path):
    state = AlertState(tmp_path / "alerts.json", ttl_seconds=3600)
    fp = "abc"
    is_new_1 = state.record_alert(fp, title="t", severity="warn")
    is_new_2 = state.record_alert(fp, title="t", severity="warn")
    assert is_new_1 is True
    assert is_new_2 is False
    raw = json.loads((tmp_path / "alerts.json").read_text())
    assert raw["alerts"][fp]["count"] == 2


def test_ttl_eviction(tmp_path: Path):
    sf = tmp_path / "alerts.json"
    fp = "old"
    initial = {
        "alerts": {
            fp: {
                "first_seen": _ts(7200),
                "last_alerted": _ts(7200),  # 2h ago, ttl 1h
                "count": 1,
                "title": "x",
                "severity": "warn",
            }
        }
    }
    sf.write_text(json.dumps(initial))
    state = AlertState(sf, ttl_seconds=3600)
    assert state.was_alerted_recently(fp) is False  # evicted on read


def test_corrupt_file_recovered(tmp_path: Path):
    sf = tmp_path / "alerts.json"
    sf.write_text("{ this is not valid json")
    state = AlertState(sf, ttl_seconds=3600)
    assert state.was_alerted_recently("anything") is False
    backups = list(tmp_path.glob("alerts.json.corrupt-*"))
    assert len(backups) == 1


def test_persistence_across_instances(tmp_path: Path):
    sf = tmp_path / "alerts.json"
    state1 = AlertState(sf, ttl_seconds=3600)
    state1.record_alert("fp1", title="t", severity="info")
    state2 = AlertState(sf, ttl_seconds=3600)
    assert state2.was_alerted_recently("fp1") is True


def test_missing_file_creates_empty_state(tmp_path: Path):
    sf = tmp_path / "subdir" / "alerts.json"
    state = AlertState(sf, ttl_seconds=3600)
    assert state.was_alerted_recently("any") is False
    state.record_alert("fp", title="t", severity="warn")
    assert sf.exists()


def _concurrent_worker(path_str: str, fp: str, q) -> None:
    from pathlib import Path as P
    from log_agent.state import AlertState as AS
    s = AS(P(path_str), ttl_seconds=3600)
    q.put(s.record_alert(fp, title="t", severity="warn"))


def test_record_alert_concurrent_writers(tmp_path: Path):
    """When two processes record the same fingerprint concurrently, exactly one
    sees is_new=True and final count is 2."""
    import multiprocessing

    sf = tmp_path / "alerts.json"
    fp = "concurrent-fp"
    ctx = multiprocessing.get_context("spawn")
    q = ctx.Queue()
    procs = [ctx.Process(target=_concurrent_worker, args=(str(sf), fp, q)) for _ in range(2)]
    for p in procs:
        p.start()
    for p in procs:
        p.join(timeout=10)
        assert p.exitcode == 0

    results = sorted([q.get(), q.get()])
    assert results == [False, True], f"expected exactly one is_new=True, got {results}"

    raw = json.loads(sf.read_text())
    assert raw["alerts"][fp]["count"] == 2
