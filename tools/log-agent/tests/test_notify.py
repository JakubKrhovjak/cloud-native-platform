import subprocess
from unittest.mock import MagicMock

from log_agent.notify import notify_macos


def test_notify_macos_calls_osascript(mocker):
    run = mocker.patch("log_agent.notify.subprocess.run", return_value=MagicMock(returncode=0))
    notify_macos(title="t", subtitle="s", message="m", play_sound=False)
    run.assert_called_once()
    args = run.call_args.args[0]
    assert args[0] == "osascript"
    assert args[1] == "-e"
    script = args[2]
    assert 'display notification "m"' in script
    assert 'with title "t"' in script
    assert 'subtitle "s"' in script
    assert "sound name" not in script


def test_notify_macos_with_sound(mocker):
    run = mocker.patch("log_agent.notify.subprocess.run", return_value=MagicMock(returncode=0))
    notify_macos(title="t", subtitle="s", message="m", play_sound=True)
    script = run.call_args.args[0][2]
    assert 'sound name "default"' in script


def test_notify_macos_escapes_double_quotes(mocker):
    run = mocker.patch("log_agent.notify.subprocess.run", return_value=MagicMock(returncode=0))
    notify_macos(title='t"x', subtitle="s", message='m"y', play_sound=False)
    script = run.call_args.args[0][2]
    # double quotes inside strings must be escaped (\\")
    assert 't\\"x' in script
    assert 'm\\"y' in script


def test_notify_macos_returns_false_on_nonzero_exit(mocker):
    mocker.patch(
        "log_agent.notify.subprocess.run",
        return_value=MagicMock(returncode=1, stderr=b"boom"),
    )
    assert notify_macos("t", "s", "m") is False


def test_notify_macos_returns_true_on_success(mocker):
    mocker.patch("log_agent.notify.subprocess.run", return_value=MagicMock(returncode=0))
    assert notify_macos("t", "s", "m") is True


def test_notify_macos_does_not_raise_when_subprocess_fails(mocker):
    mocker.patch("log_agent.notify.subprocess.run", side_effect=FileNotFoundError("osascript"))
    assert notify_macos("t", "s", "m") is False
