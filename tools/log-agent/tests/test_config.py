from pathlib import Path
import pytest
from log_agent.config import Config, load_config


def test_load_config_defaults(tmp_path: Path):
    cfg_file = tmp_path / "config.yaml"
    cfg_file.write_text(
        "namespaces: [apps, infra]\n"
        "model: claude-sonnet-4-6\n"
        "periodic:\n"
        "  lookback_seconds: 900\n"
        "  max_iterations: 25\n"
        "alerts:\n"
        "  dedup_ttl_seconds: 21600\n"
        "  prefilter_pattern: \"ERROR|panic\"\n"
        "limits:\n"
        "  max_log_lines: 2000\n"
        "  max_session_input_tokens: 50000\n"
    )
    cfg = load_config(cfg_file)
    assert isinstance(cfg, Config)
    assert cfg.namespaces == ["apps", "infra"]
    assert cfg.model == "claude-sonnet-4-6"
    assert cfg.periodic.lookback_seconds == 900
    assert cfg.periodic.max_iterations == 25
    assert cfg.alerts.dedup_ttl_seconds == 21600
    assert cfg.alerts.prefilter_pattern == "ERROR|panic"
    assert cfg.limits.max_log_lines == 2000
    assert cfg.limits.max_session_input_tokens == 50000


def test_load_config_missing_file_raises(tmp_path: Path):
    with pytest.raises(FileNotFoundError):
        load_config(tmp_path / "nope.yaml")


def test_load_config_invalid_yaml_raises(tmp_path: Path):
    cfg_file = tmp_path / "config.yaml"
    cfg_file.write_text("this is: not: valid: yaml:\n  - x\n")
    with pytest.raises(ValueError):
        load_config(cfg_file)
