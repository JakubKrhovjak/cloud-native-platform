import re


def compile_pattern(pattern: str) -> re.Pattern[str]:
    return re.compile(pattern)


def match_lines(pattern: re.Pattern[str], lines: list[str]) -> list[tuple[int, str]]:
    return [(i, line) for i, line in enumerate(lines) if pattern.search(line)]
