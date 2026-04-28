import fcntl
import hashlib
import json
import os
import sys
import time
from contextlib import contextmanager
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def fingerprint(namespace: str, pod: str, error_signature: str) -> str:
    raw = f"{namespace}|{pod}|{error_signature}".encode("utf-8")
    return hashlib.sha256(raw).hexdigest()


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _parse_iso(s: str) -> datetime:
    return datetime.fromisoformat(s)


class AlertState:
    def __init__(self, path: Path, ttl_seconds: int) -> None:
        self.path = path
        self.ttl_seconds = ttl_seconds
        self.path.parent.mkdir(parents=True, exist_ok=True)
        # Dedicated lock file — its inode never changes, so locks are always comparable.
        self._lock_path = path.with_name(path.name + ".lock")

    @contextmanager
    def _locked(self, mode: str):
        # mode: "shared" or "exclusive"
        # Lock is acquired on a dedicated .lock file whose inode never changes.
        # The data file may be replaced atomically via os.replace() without
        # invalidating the held lock.
        flock_op = fcntl.LOCK_SH if mode == "shared" else fcntl.LOCK_EX
        fd = os.open(self._lock_path, os.O_RDWR | os.O_CREAT, 0o644)
        try:
            fcntl.flock(fd, flock_op)
            try:
                yield
            finally:
                fcntl.flock(fd, fcntl.LOCK_UN)
        finally:
            os.close(fd)

    def _read_data(self) -> dict[str, Any]:
        if not self.path.exists():
            return {"alerts": {}}
        try:
            raw = self.path.read_text()
        except OSError:
            return {"alerts": {}}
        if not raw.strip():
            return {"alerts": {}}
        try:
            return json.loads(raw)
        except json.JSONDecodeError:
            backup = self.path.with_name(f"{self.path.name}.corrupt-{int(time.time())}")
            self.path.rename(backup)
            print(
                f"log-agent: corrupt state file recovered: {self.path} -> {backup.name}",
                file=sys.stderr,
            )
            return {"alerts": {}}

    def _write_data_atomic(self, data: dict[str, Any]) -> None:
        # Atomic write via temp + rename. Caller must hold exclusive lock.
        tmp = self.path.with_name(self.path.name + ".tmp")
        tmp.write_text(json.dumps(data, indent=2))
        os.replace(tmp, self.path)

    def _evict_expired(self, data: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now(timezone.utc)
        kept: dict[str, Any] = {}
        for fp, rec in data.get("alerts", {}).items():
            try:
                age = (now - _parse_iso(rec["last_alerted"])).total_seconds()
            except (KeyError, ValueError, TypeError):
                continue  # drop malformed records
            if age < self.ttl_seconds:
                kept[fp] = rec
        return {"alerts": kept}

    def was_alerted_recently(self, fp: str) -> bool:
        with self._locked("shared"):
            data = self._evict_expired(self._read_data())
        return fp in data["alerts"]

    def record_alert(self, fp: str, title: str, severity: str) -> bool:
        """Returns True if this is the first alert within TTL (notification should fire)."""
        with self._locked("exclusive"):
            data = self._evict_expired(self._read_data())
            is_new = fp not in data["alerts"]
            now = _now_iso()
            if is_new:
                data["alerts"][fp] = {
                    "first_seen": now,
                    "last_alerted": now,
                    "count": 1,
                    "title": title,
                    "severity": severity,
                }
            else:
                rec = data["alerts"][fp]
                rec["last_alerted"] = now
                rec["count"] += 1
                rec["title"] = title
                rec["severity"] = severity
            self._write_data_atomic(data)
        return is_new
