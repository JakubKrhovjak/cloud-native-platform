from dataclasses import dataclass
from pathlib import Path

import yaml


@dataclass
class PeriodicConfig:
    lookback_seconds: int
    max_iterations: int


@dataclass
class AlertsConfig:
    dedup_ttl_seconds: int
    prefilter_pattern: str


@dataclass
class LimitsConfig:
    max_log_lines: int
    max_session_input_tokens: int


@dataclass
class Config:
    namespaces: list[str]
    model: str
    periodic: PeriodicConfig
    alerts: AlertsConfig
    limits: LimitsConfig


def load_config(path: Path) -> Config:
    if not path.exists():
        raise FileNotFoundError(f"config file not found: {path}")
    try:
        raw = yaml.safe_load(path.read_text())
    except yaml.YAMLError as e:
        raise ValueError(f"invalid YAML in {path}: {e}") from e
    if not isinstance(raw, dict):
        raise ValueError(f"config root must be a mapping, got {type(raw).__name__}")
    try:
        return Config(
            namespaces=list(raw["namespaces"]),
            model=str(raw["model"]),
            periodic=PeriodicConfig(**raw["periodic"]),
            alerts=AlertsConfig(**raw["alerts"]),
            limits=LimitsConfig(**raw["limits"]),
        )
    except (KeyError, TypeError) as e:
        raise ValueError(f"invalid config structure: {e}") from e
