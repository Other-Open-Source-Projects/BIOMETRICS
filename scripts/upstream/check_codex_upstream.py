#!/usr/bin/env python3
"""Check codex upstream release lag against repository policy."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path


DEFAULT_LOCKFILE = Path("third_party/codex-upstream/upstream.lock.json")


@dataclass
class Release:
    tag: str
    published_at: datetime
    html_url: str


def parse_timestamp(raw: str) -> datetime:
    return datetime.fromisoformat(raw.replace("Z", "+00:00")).astimezone(UTC)


def format_timestamp(ts: datetime) -> str:
    return ts.astimezone(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def fetch_latest_release(repo: str, tag_prefix: str, include_prereleases: bool) -> Release:
    url = f"https://api.github.com/repos/{repo}/releases?per_page=50"
    req = urllib.request.Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "User-Agent": "biometrics-upstream-watch/1.0",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            payload = json.load(resp)
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"GitHub API error ({exc.code}): {exc.reason}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"Unable to reach GitHub API: {exc.reason}") from exc

    if not isinstance(payload, list):
        raise RuntimeError("Unexpected GitHub API response for releases list")

    for release in payload:
        if release.get("draft"):
            continue
        if not include_prereleases and release.get("prerelease"):
            continue
        tag = release.get("tag_name", "")
        if tag_prefix and not tag.startswith(tag_prefix):
            continue
        try:
            return Release(
                tag=tag,
                published_at=parse_timestamp(release["published_at"]),
                html_url=release["html_url"],
            )
        except KeyError as exc:
            raise RuntimeError(f"Unexpected release entry, missing field: {exc}") from exc

    raise RuntimeError(
        f"No releases matched repo={repo}, tag_prefix={tag_prefix!r}, include_prereleases={include_prereleases}"
    )


def load_lockfile(path: Path) -> dict:
    if not path.exists():
        raise RuntimeError(f"Missing lockfile: {path}")
    return json.loads(path.read_text(encoding="utf-8"))


def write_lockfile(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def write_summary(path: Path, lines: list[str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def build_summary(
    *,
    repo: str,
    tracked_tag: str,
    tracked_at: datetime,
    latest: Release,
    check_time: datetime,
    max_lag_days: int,
    status: str,
    reason: str,
    tag_prefix: str,
    include_prereleases: bool,
) -> list[str]:
    hours_since_latest = (check_time - latest.published_at).total_seconds() / 3600
    return [
        "## Codex Upstream Watch",
        "",
        f"- Status: **{status}**",
        f"- Reason: {reason}",
        f"- Repository: `{repo}`",
        f"- Release selector: `tag_prefix={tag_prefix or '*'}, include_prereleases={include_prereleases}`",
        f"- Tracked release: `{tracked_tag}` ({format_timestamp(tracked_at)})",
        f"- Latest release: `{latest.tag}` ({format_timestamp(latest.published_at)})",
        f"- Latest release URL: {latest.html_url}",
        f"- Hours since latest release: `{hours_since_latest:.2f}`",
        f"- Max allowed lag window: `{max_lag_days}` days",
        f"- Check timestamp (UTC): `{format_timestamp(check_time)}`",
    ]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default="openai/codex", help="GitHub repository owner/name")
    parser.add_argument("--lockfile", type=Path, default=DEFAULT_LOCKFILE)
    parser.add_argument("--max-lag-days", type=int, default=7)
    parser.add_argument("--tag-prefix", default=None)
    parser.add_argument("--include-prereleases", action="store_true", default=None)
    parser.add_argument("--update-lock", action="store_true")
    parser.add_argument("--summary-file", type=Path, default=Path("/tmp/codex-upstream-summary.md"))
    args = parser.parse_args()

    now = datetime.now(tz=UTC)
    lock = load_lockfile(args.lockfile)

    tag_prefix = args.tag_prefix
    if tag_prefix is None:
        tag_prefix = str(lock.get("tag_prefix", ""))

    include_prereleases = args.include_prereleases
    if include_prereleases is None:
        include_prereleases = bool(lock.get("include_prereleases", False))

    latest = fetch_latest_release(args.repo, tag_prefix, include_prereleases)

    tracked_tag = lock.get("tracked_release_tag")
    tracked_published_at_raw = lock.get("tracked_release_published_at")
    if not tracked_tag or not tracked_published_at_raw:
        raise RuntimeError(
            f"Lockfile {args.lockfile} must contain tracked_release_tag and tracked_release_published_at"
        )

    tracked_published_at = parse_timestamp(tracked_published_at_raw)
    release_window_days = (now - latest.published_at).total_seconds() / 86400
    in_window = release_window_days <= float(args.max_lag_days)

    if tracked_tag == latest.tag:
        status = "PASS"
        reason = "Tracked release matches upstream latest release."
    elif in_window:
        status = "WARN"
        reason = (
            "Tracked release is behind upstream, but still inside the allowed lag window. "
            "Open a sync PR before the window expires."
        )
    else:
        status = "FAIL"
        reason = (
            "Tracked release is behind upstream and outside the allowed lag window. "
            "A codex-upstream sync PR is now mandatory."
        )

    if args.update_lock:
        lock.update(
            {
                "source_repo": args.repo,
                "tracked_release_tag": latest.tag,
                "tracked_release_published_at": format_timestamp(latest.published_at),
                "last_checked_at": format_timestamp(now),
                "max_release_lag_days": args.max_lag_days,
                "tag_prefix": tag_prefix,
                "include_prereleases": include_prereleases,
            }
        )
        write_lockfile(args.lockfile, lock)
        tracked_tag = latest.tag
        tracked_published_at = latest.published_at
        status = "PASS"
        reason = "Lockfile updated to upstream latest release."

    summary_lines = build_summary(
        repo=args.repo,
        tracked_tag=tracked_tag,
        tracked_at=tracked_published_at,
        latest=latest,
        check_time=now,
        max_lag_days=args.max_lag_days,
        status=status,
        reason=reason,
        tag_prefix=tag_prefix,
        include_prereleases=include_prereleases,
    )
    write_summary(args.summary_file, summary_lines)
    print("\n".join(summary_lines))

    return 1 if status == "FAIL" else 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except RuntimeError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
