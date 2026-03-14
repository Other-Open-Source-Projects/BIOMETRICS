#!/usr/bin/env python3
"""Inject latest eval evidence into docs/releases/APEX_SCOREBOARD.md."""

from __future__ import annotations

import argparse
import json
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--results",
        required=True,
        help="Path to eval-results-*.json",
    )
    parser.add_argument(
        "--report",
        default="docs/releases/APEX_SCOREBOARD.md",
        help="Path to APEX_SCOREBOARD.md",
    )
    return parser.parse_args()


def pct(value: float) -> str:
    return f"{value:.2f}%"


def main() -> int:
    args = parse_args()
    results_path = Path(args.results).resolve()
    report_path = Path(args.report)

    payload = json.loads(results_path.read_text(encoding="utf-8"))
    candidate = payload.get("candidate_metrics", {})
    baseline = payload.get("baseline_metrics", {})
    comparison = payload.get("comparison", {})
    candidate_strategy = payload.get("candidate_strategy", "unknown")
    baseline_strategy = payload.get("baseline_strategy", "unknown")
    regression = bool(payload.get("regression_detected", False))
    eval_run_id = payload.get("eval_run_id", "unknown")

    baseline_cmp = comparison.get(baseline_strategy, {})
    time_improvement = float(baseline_cmp.get("time_to_green_improvement_percent", 0.0))
    cost_improvement = float(baseline_cmp.get("cost_improvement_percent", 0.0))
    quality = float(candidate.get("quality_score", 0.0))

    gates = {
        "quality>=0.90": quality >= 0.90,
        "time_improvement>=25%": time_improvement >= 25.0,
        "cost_improvement>=20%": cost_improvement >= 20.0,
        "regression_detected=false": not regression,
    }
    overall = "PASS" if all(gates.values()) else "FAIL"

    competitor_lines = []
    for strategy, entry in sorted(comparison.items()):
        competitor_lines.append(
            f"- {strategy}: quality_delta={entry.get('quality_delta', 0.0):.4f}, "
            f"time_improvement={pct(float(entry.get('time_to_green_improvement_percent', 0.0)))}, "
            f"cost_improvement={pct(float(entry.get('cost_improvement_percent', 0.0)))}, "
            f"composite_delta={entry.get('composite_delta', 0.0):.4f}"
        )

    auto_block = "\n".join(
        [
            "<!-- BEGIN AUTO_APEX -->",
            "## Latest Apex Evidence",
            f"- Eval run id: `{eval_run_id}`",
            f"- Results file: `{results_path}`",
            f"- Candidate strategy: `{candidate_strategy}`",
            f"- Baseline strategy: `{baseline_strategy}`",
            f"- Candidate quality_score: {quality:.4f}",
            f"- Candidate median_time_to_green_seconds: {float(candidate.get('median_time_to_green_seconds', 0.0)):.2f}",
            f"- Candidate cost_per_success: {float(candidate.get('cost_per_success', 0.0)):.6f}",
            f"- Baseline quality_score: {float(baseline.get('quality_score', 0.0)):.4f}",
            f"- Baseline median_time_to_green_seconds: {float(baseline.get('median_time_to_green_seconds', 0.0)):.2f}",
            f"- Baseline cost_per_success: {float(baseline.get('cost_per_success', 0.0)):.6f}",
            f"- vs baseline time improvement: {pct(time_improvement)}",
            f"- vs baseline cost improvement: {pct(cost_improvement)}",
            f"- regression_detected: `{str(regression).lower()}`",
            "",
            "### Comparison Table",
            *competitor_lines,
            "",
            "### Gate Status",
            *[f"- {name}: {'PASS' if passed else 'FAIL'}" for name, passed in gates.items()],
            f"- overall: **{overall}**",
            "<!-- END AUTO_APEX -->",
        ]
    )

    content = report_path.read_text(encoding="utf-8")
    begin_marker = "<!-- BEGIN AUTO_APEX -->"
    end_marker = "<!-- END AUTO_APEX -->"
    if begin_marker in content and end_marker in content:
        prefix = content.split(begin_marker, 1)[0]
        suffix = content.split(end_marker, 1)[1]
        content = prefix + auto_block + suffix
    else:
        content = content.rstrip() + "\n\n" + auto_block + "\n"

    report_path.write_text(content, encoding="utf-8")
    print(f"Updated {report_path} from {results_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
