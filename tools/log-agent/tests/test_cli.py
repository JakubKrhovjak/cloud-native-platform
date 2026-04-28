from pathlib import Path
from unittest.mock import MagicMock

import pytest

from log_agent import cli


@pytest.fixture
def fake_env(monkeypatch, tmp_path):
    monkeypatch.setenv("ANTHROPIC_API_KEY", "test-key")
    cfg_path = tmp_path / "config.yaml"
    cfg_path.write_text(
        "namespaces: [apps]\n"
        "model: claude-sonnet-4-6\n"
        "periodic:\n  lookback_seconds: 900\n  max_iterations: 25\n"
        "alerts:\n  dedup_ttl_seconds: 21600\n  prefilter_pattern: \"ERROR\"\n"
        "limits:\n  max_log_lines: 2000\n  max_session_input_tokens: 50000\n"
    )
    state_dir = tmp_path / "state"
    monkeypatch.setenv("LOG_AGENT_CONFIG", str(cfg_path))
    monkeypatch.setenv("LOG_AGENT_STATE_DIR", str(state_dir))
    return cfg_path, state_dir


def test_cli_missing_api_key_exits(monkeypatch, capsys):
    monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)
    with pytest.raises(SystemExit) as exc:
        cli.main(["check"])
    assert exc.value.code == 1
    assert "ANTHROPIC_API_KEY" in capsys.readouterr().err


def test_cli_check_subcommand_runs_agent(fake_env, mocker):
    fake_agent = MagicMock()
    fake_agent.run.return_value = MagicMock(text="ok", aborted=False, iterations=1)
    fake_agent.ctx = MagicMock()
    fake_agent.ctx.config = MagicMock()
    fake_agent.ctx.config.periodic = MagicMock(lookback_seconds=900)
    mocker.patch("log_agent.cli._build_agent", return_value=fake_agent)
    rc = cli.main(["check"])
    assert rc == 0
    fake_agent.run.assert_called_once()
    assert "periodic check" in fake_agent.run.call_args.args[0].lower()


def test_cli_ask_subcommand_prints_response(fake_env, mocker, capsys):
    fake_agent = MagicMock()
    fake_agent.run.return_value = MagicMock(text="answer text", aborted=False, iterations=2)
    mocker.patch("log_agent.cli._build_agent", return_value=fake_agent)
    rc = cli.main(["ask", "what happened"])
    assert rc == 0
    assert "answer text" in capsys.readouterr().out
    assert fake_agent.run.call_args.args[0] == "what happened"


def test_cli_chat_subcommand_loops_until_eof(fake_env, mocker, capsys):
    fake_agent = MagicMock()
    fake_agent.run.side_effect = [
        MagicMock(text="r1", aborted=False, iterations=0),
        MagicMock(text="r2", aborted=False, iterations=0),
    ]
    mocker.patch("log_agent.cli._build_agent", return_value=fake_agent)
    inputs = iter(["hello", "again", EOFError()])

    def fake_input(prompt=""):
        v = next(inputs)
        if isinstance(v, BaseException):
            raise v
        return v

    mocker.patch("builtins.input", side_effect=fake_input)
    rc = cli.main(["chat"])
    assert rc == 0
    out = capsys.readouterr().out
    assert "r1" in out
    assert "r2" in out
