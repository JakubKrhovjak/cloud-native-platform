import logging
import subprocess

log = logging.getLogger(__name__)


def _escape(s: str) -> str:
    return s.replace("\\", "\\\\").replace('"', '\\"')


def notify_macos(title: str, subtitle: str, message: str, play_sound: bool = False) -> bool:
    parts = [
        f'display notification "{_escape(message)}"',
        f'with title "{_escape(title)}"',
        f'subtitle "{_escape(subtitle)}"',
    ]
    if play_sound:
        parts.append('sound name "default"')
    script = " ".join(parts)
    try:
        result = subprocess.run(
            ["osascript", "-e", script],
            capture_output=True,
            timeout=5,
        )
        if result.returncode != 0:
            log.warning("osascript failed: %s", result.stderr.decode("utf-8", "ignore"))
            return False
        return True
    except (FileNotFoundError, subprocess.TimeoutExpired) as e:
        log.warning("notify_macos failed: %s", e)
        return False
