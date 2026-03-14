#!/usr/bin/env python3
"""Validate soak summary against release gates."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Analyze BIOMETRICS soak summary")
    parser.add_argument("--summary", required=True, help="Path to soak summary JSON")
    parser.add_argument("--min-success-rate", type=float, default=0.98, help="Minimum required run success rate")
    parser.add_argument("--max-timeouts", type=int, default=0, help="Maximum allowed timed-out runs")
    parser.add_argument("--max-dispatch-p95-ms", type=float, default=250.0, help="Maximum allowed dispatch latency p95 estimate in ms")
    parser.add_argument("--max-fallback-rate", type=float, default=0.05, help="Maximum allowed fallback rate per run")
    parser.add_argument("--max-backpressure-per-run", type=float, default=20.0, help="Maximum allowed average backpressure signals per run")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    summary_path = Path(args.summary)
    if not summary_path.exists():
        print(f"ERROR: summary file not found: {summary_path}", file=sys.stderr)
        return 2

    data = json.loads(summary_path.read_text(encoding="utf-8"))
    success_rate = float(data.get("run_success_rate", 0.0))
    timeouts = int(data.get("runs_timed_out", 0))
    dispatch_p95_ms = float(data.get("dispatch_latency_p95_estimate_ms", 0.0))
    fallback_rate = float(data.get("run_fallback_rate", 0.0))
    backpressure_per_run = float(data.get("run_backpressure_per_run", 0.0))

    checks = [
        ("success_rate", success_rate >= args.min_success_rate, f"{success_rate:.4f} >= {args.min_success_rate:.4f}"),
        ("timeouts", timeouts <= args.max_timeouts, f"{timeouts} <= {args.max_timeouts}"),
        ("dispatch_p95_ms", dispatch_p95_ms <= args.max_dispatch_p95_ms, f"{dispatch_p95_ms:.2f} <= {args.max_dispatch_p95_ms:.2f}"),
        ("fallback_rate", fallback_rate <= args.max_fallback_rate, f"{fallback_rate:.4f} <= {args.max_fallback_rate:.4f}"),
        (
            "backpressure_per_run",
            backpressure_per_run <= args.max_backpressure_per_run,
            f"{backpressure_per_run:.2f} <= {args.max_backpressure_per_run:.2f}",
        ),
    ]

    failed = [name for name, ok, _ in checks if not ok]
    for name, ok, detail in checks:
        print(f"{name}: {'PASS' if ok else 'FAIL'} ({detail})")

    if failed:
        print(f"Gate check failed: {', '.join(failed)}", file=sys.stderr)
        return 1

    print("All soak gates passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
