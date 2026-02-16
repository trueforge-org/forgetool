#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import subprocess
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
COVERAGE_FILE = ROOT / "coverage.txt"
CHECKLIST_FILE = ROOT / "coverage_checklist.md"

COVERAGE_LINE_PATTERN = re.compile(r"(.+?):\d+\.\d+,\d+\.\d+\s+(\d+)\s+(\d+)$")
MODULE_PREFIX_PATTERN = re.compile(r"^github\.com/trueforge-org/forgetool/")


def run_tests_for_coverage() -> None:
    package_list_result = subprocess.run(
        ["go", "list", "-f", "{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}", "./..."],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )

    packages = [line.strip() for line in package_list_result.stdout.splitlines() if line.strip()]
    if not packages:
        raise RuntimeError("No Go packages with tests were found.")

    subprocess.run(
        ["go", "test", "-covermode=atomic", f"-coverprofile={COVERAGE_FILE.name}", *packages],
        cwd=ROOT,
        check=True,
    )


def parse_coverage_profile() -> list[tuple[float, int, int, str]]:
    if not COVERAGE_FILE.exists():
        raise FileNotFoundError(
            f"Coverage profile not found: {COVERAGE_FILE}. Run tests first or omit --from-existing."
        )

    stats: dict[str, list[int]] = defaultdict(lambda: [0, 0])

    for raw_line in COVERAGE_FILE.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("mode:"):
            continue

        match = COVERAGE_LINE_PATTERN.match(line)
        if not match:
            continue

        file_path, statements, count = match.group(1), int(match.group(2)), int(match.group(3))
        file_path = MODULE_PREFIX_PATTERN.sub("", file_path)

        stats[file_path][1] += statements
        if count > 0:
            stats[file_path][0] += statements

    rows: list[tuple[float, int, int, str]] = []
    for file_path, (covered, total) in stats.items():
        if total == 0:
            continue
        rows.append((covered * 100 / total, covered, total, file_path))

    rows.sort(key=lambda row: (row[0], row[3]))
    return rows


def detect_checkbox_state() -> str:
    if not CHECKLIST_FILE.exists():
        return "x"

    for raw_line in CHECKLIST_FILE.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if line.startswith("- [ ]"):
            return " "
        if line.startswith("- [x]"):
            return "x"

    return "x"


def write_checklist(rows: list[tuple[float, int, int, str]]) -> None:
    checkbox_state = detect_checkbox_state()

    lines = [
        "# Test Coverage Checklist",
        "",
        "Sorted from lowest to highest coverage.",
        "",
    ]

    for pct, covered, total, file_path in rows:
        lines.append(f"- [{checkbox_state}] {pct:6.2f}% ({covered}/{total}) - {file_path}")

    CHECKLIST_FILE.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Update coverage_checklist.md from coverage.txt (optionally regenerating coverage.txt)."
    )
    parser.add_argument(
        "--from-existing",
        action="store_true",
        help="Do not run tests; only parse existing coverage.txt.",
    )
    args = parser.parse_args()

    if not args.from_existing:
        run_tests_for_coverage()

    rows = parse_coverage_profile()
    write_checklist(rows)

    print(f"Updated {CHECKLIST_FILE.name} with {len(rows)} entries.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
