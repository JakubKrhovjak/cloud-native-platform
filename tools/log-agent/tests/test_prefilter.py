import re
from log_agent.prefilter import compile_pattern, match_lines


PATTERN = r"ERROR|panic|OOM|FATAL|HTTP 5\d{2}"


def test_compile_pattern_returns_pattern():
    p = compile_pattern(PATTERN)
    assert isinstance(p, re.Pattern)


def test_match_lines_finds_error():
    p = compile_pattern(PATTERN)
    lines = [
        "2026-04-28T10:00:00 INFO starting up",
        "2026-04-28T10:00:01 ERROR connection refused",
        "2026-04-28T10:00:02 INFO healthy",
    ]
    matched = match_lines(p, lines)
    assert matched == [(1, "2026-04-28T10:00:01 ERROR connection refused")]


def test_match_lines_finds_multiple():
    p = compile_pattern(PATTERN)
    lines = [
        "INFO ok",
        "panic: nil pointer",
        "OOMKilled by kubelet",
        "INFO recovered",
        "GET /x HTTP 502 Bad Gateway",
    ]
    matched = match_lines(p, lines)
    assert [m[0] for m in matched] == [1, 2, 4]


def test_match_lines_empty_input():
    p = compile_pattern(PATTERN)
    assert match_lines(p, []) == []


def test_match_lines_no_matches():
    p = compile_pattern(PATTERN)
    lines = ["INFO 1", "DEBUG 2", "WARN 3"]
    assert match_lines(p, lines) == []


def test_compile_pattern_invalid_raises():
    import pytest
    with pytest.raises(re.error):
        compile_pattern("(unclosed")
