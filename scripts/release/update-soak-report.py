#!/usr/bin/env python3
"""Inject soak summary metrics into docs/releases/SOAK_72H_REPORT.md."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

START_MARKER = "<!-- BEGIN AUTO_RESULTS -->"
END_MARKER = "<!-- END AUTO_RESULTS -->"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Update soak report from summary JSON")
    parser.add_argument("--summary", required=True, help="Path to soak summary JSON")
    parser.add_argument(
        "--report",
        default="docs/releases/SOAK_72H_REPORT.md",
        help="Path to SOAK_72H_REPORT.md",
    )
    parser.add_argument("--window-start", default="2026-03-19", help="Soak window start date")
    parser.add_argument("--window-end", default="2026-03-22", help="Soak window end date")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    summary_path = Path(args.summary)
    report_path = Path(args.report)

    if not summary_path.exists():
        raise SystemExit(f"Summary not found: {summary_path}")
    if not report_path.exists():
        raise SystemExit(f"Report not found: {report_path}")

    summary = json.loads(summary_path.read_text(encoding="utf-8"))

    success_rate = float(summary.get("run_success_rate", 0.0))
    timed_out = int(summary.get("runs_timed_out", 0))
    dispatch_p95 = float(summary.get("dispatch_latency_p95_estimate_ms", 0.0))
    fallback_rate = float(summary.get("run_fallback_rate", 0.0))
    backpressure_per_run = float(summary.get("run_backpressure_per_run", 0.0))
    profile_label = str(summary.get("profile_label", ""))
    configured_duration = int(summary.get("duration_seconds", 0))
    actual_duration = int(summary.get("harness_elapsed_seconds", 0))
    records_window_seconds = int(summary.get("records_window_seconds", 0))
    gate_b_qualified = profile_label == "ga-72h" and configured_duration >= 72 * 3600

    gates = summary.get("gates", {})
    thresholds_passed = all(
        bool(gates.get(key, False))
        for key in (
            "success_rate_gte_98",
            "no_timeouts",
            "dispatch_p95_lte_250",
            "fallback_rate_lte_0_05",
            "backpressure_per_run_lte_20",
        )
    )
    passed = thresholds_passed and gate_b_qualified
    run_class = "GA qualifying" if gate_b_qualified else "Rehearsal / non-GA"

    block = f"""{START_MARKER}
## Auto-Generated Results ({summary_path.name})
- Window: {args.window_start} to {args.window_end}
- Class: {run_class}
- Profile label: {profile_label or "unspecified"}
- Configured duration seconds: {configured_duration}
- Actual harness duration seconds: {actual_duration}
- Records window seconds: {records_window_seconds}
- Runs started: {summary.get('runs_started', 0)}
- Runs completed: {summary.get('runs_completed', 0)}
- Runs failed: {summary.get('runs_failed', 0)}
- Runs timed out: {timed_out}
- Success rate: {success_rate:.4f}
- Dispatch p95 (ms): {dispatch_p95:.2f}
- Fallback rate: {fallback_rate:.4f}
- Backpressure per run: {backpressure_per_run:.2f}
- Queue depth max: {float(summary.get('ready_queue_depth_max', 0.0)):.2f}
- Records file: `{summary.get('records_file', '')}`
- Baseline metrics file: `{summary.get('baseline_metrics_file', '')}`
- Metrics file: `{summary.get('metrics_file', '')}`
- Summary file: `{summary_path}`

### Metrics Delta (Soak Window)
- Dispatch count: {summary.get('metrics_delta', {}).get('biometrics_task_dispatch_latency_count', 0)}
- Backpressure signals: {summary.get('metrics_delta', {}).get('biometrics_backpressure_signals', 0)}
- Fallbacks triggered: {summary.get('metrics_delta', {}).get('biometrics_fallbacks_triggered', 0)}

### Auto Decision
- Thresholds: {'PASS' if thresholds_passed else 'FAIL'}
- GA qualification: {'PASS' if gate_b_qualified else 'FAIL'}
- Overall: {'PASS' if passed else 'FAIL'}
{END_MARKER}"""

    content = report_path.read_text(encoding="utf-8")
    if START_MARKER in content and END_MARKER in content:
        prefix = content.split(START_MARKER, 1)[0]
        suffix = content.split(END_MARKER, 1)[1]
        updated = prefix + block + suffix
    else:
        updated = content.rstrip() + "\n\n" + block + "\n"

    report_path.write_text(updated, encoding="utf-8")
    print(f"Updated {report_path} using {summary_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
