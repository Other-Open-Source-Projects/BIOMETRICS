#!/usr/bin/env python3

import argparse
import base64
import json
import os
import re
import shlex
import subprocess
import sys
import urllib.request
from datetime import datetime, timezone
from pathlib import Path


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def read_env(key: str, default: str = "") -> str:
    return os.environ.get(key, default).strip()


def run_upload_cmd(upload_cmd: str, video_path: str) -> str:
    argv = shlex.split(upload_cmd)
    argv.append(video_path)
    proc = subprocess.run(argv, capture_output=True, text=True)
    if proc.returncode != 0:
        raise RuntimeError(
            f"upload cmd failed rc={proc.returncode} stderr={proc.stderr.strip()[:400]}"
        )
    url = (proc.stdout or "").strip().splitlines()[-1].strip() if proc.stdout else ""
    if not url:
        raise RuntimeError("upload cmd returned empty url")
    return url


def nim_chat_completion(
    base_url: str,
    api_key: str,
    model: str,
    prompt: str,
    video_url: str,
    media_type: str,
    video_fps: float | None,
) -> dict:
    url = base_url.rstrip("/") + "/chat/completions"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}",
    }
    payload: dict = {
        "model": model,
        "messages": [
            {
                "role": "user",
                "content": [
                    {"type": "text", "text": prompt},
                    {},
                ],
            }
        ],
    }

    media_type = (media_type or "").strip().lower()
    if media_type == "video_url":
        payload["messages"][0]["content"][1] = {
            "type": "video_url",
            "video_url": {"url": video_url},
        }
    else:
        payload["messages"][0]["content"][1] = {
            "type": "file",
            "url": video_url,
        }
    if video_fps is not None:
        payload["extra_body"] = {
            "media_io_kwargs": {"video": {"fps": float(video_fps)}}
        }

    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers=headers,
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=180) as resp:
        raw = resp.read().decode("utf-8")
    return json.loads(raw)


def extract_message_content(resp: dict) -> str:
    choices = resp.get("choices") or []
    if not choices:
        return ""
    msg = (choices[0] or {}).get("message") or {}
    content = msg.get("content")
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts = []
        for item in content:
            if isinstance(item, dict) and item.get("type") == "text":
                parts.append(str(item.get("text", "")))
        return "\n".join(parts).strip()
    return ""


def transcode_for_validation(source: Path, dest: Path) -> None:
    cmd = [
        "ffmpeg",
        "-nostdin",
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-i",
        str(source),
        "-vf",
        "scale=1280:-2,fps=4",
        "-c:v",
        "libx264",
        "-preset",
        "veryfast",
        "-crf",
        "30",
        "-pix_fmt",
        "yuv420p",
        str(dest),
    ]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        raise RuntimeError(
            f"ffmpeg transcode failed rc={proc.returncode} stderr={proc.stderr.strip()[:400]}"
        )


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--video", required=True)
    ap.add_argument("--task", required=True)
    ap.add_argument("--out", required=True)
    args = ap.parse_args()

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    validate_enabled = read_env("VISUAL_TRUTH_VALIDATE", "0") == "1"
    allow_upload = read_env("VISUAL_TRUTH_ALLOW_UPLOAD", "0") == "1"
    allow_base64 = read_env("VISUAL_TRUTH_ALLOW_BASE64", "0") == "1"
    strict = read_env("VISUAL_TRUTH_VALIDATE_STRICT", "1") == "1"

    result: dict = {
        "at": now_iso(),
        "validated": False,
        "task": args.task,
        "video": str(args.video),
        "status": "skipped" if not validate_enabled else "pending",
    }

    if not validate_enabled:
        out_path.write_text(
            json.dumps(result, indent=2, ensure_ascii=True), encoding="utf-8"
        )
        return 0

    base_url = read_env("NIM_BASE_URL", "https://integrate.api.nvidia.com/v1")
    api_key = read_env("NIM_API_KEY", "") or read_env("NVIDIA_API_KEY", "")
    model = read_env("NIM_MODEL", "nvidia/cosmos-reason2-8b")
    media_type = read_env("VISUAL_TRUTH_NIM_MEDIA_TYPE", "file")
    video_fps_raw = read_env("VISUAL_TRUTH_VALIDATE_FPS", "4")
    try:
        video_fps = float(video_fps_raw) if video_fps_raw else None
    except Exception:
        video_fps = None

    if not api_key:
        result["status"] = "error"
        result["error"] = "missing NIM_API_KEY or NVIDIA_API_KEY"
        out_path.write_text(
            json.dumps(result, indent=2, ensure_ascii=True), encoding="utf-8"
        )
        return 2 if strict else 0

    video_path = Path(args.video)
    if not video_path.exists():
        result["status"] = "error"
        result["error"] = "video not found"
        out_path.write_text(
            json.dumps(result, indent=2, ensure_ascii=True), encoding="utf-8"
        )
        return 2 if strict else 0

    prompt = (
        "Analysiere diesen Videoabschnitt des gesamten Desktops.\n"
        f"Task: {args.task}\n"
        "Pruefe:\n"
        "1. Wurde die Interaktion korrekt ausgefuehrt?\n"
        "2. Erscheinen Fehlermeldungen im Terminal oder im Browser?\n\n"
        "Antworte genau mit einem der folgenden Formate:\n"
        "VALIDATED\n"
        "ERROR:<kurzer Grund>\n"
    )

    try:
        upload_cmd = read_env("VISUAL_TRUTH_UPLOAD_CMD", "")
        video_url = ""
        if allow_upload and upload_cmd:
            video_url = run_upload_cmd(upload_cmd, str(video_path))
            result["upload"] = {"method": "cmd", "cmd": upload_cmd}
        elif allow_base64 and allow_upload:
            max_bytes = int(
                read_env("VISUAL_TRUTH_MAX_BASE64_BYTES", "8000000") or "8000000"
            )
            encode_path = video_path
            size = encode_path.stat().st_size
            if size > max_bytes:
                candidate = video_path.with_name(video_path.stem + ".validate.mp4")
                transcode_for_validation(video_path, candidate)
                encode_path = candidate
                size = encode_path.stat().st_size
                result["upload"] = {
                    "method": "base64",
                    "bytes": size,
                    "transcoded": True,
                    "source": str(video_path),
                    "derived": str(candidate),
                }
            if size > max_bytes:
                raise RuntimeError(
                    f"video too large for base64 after transcode ({size} bytes > {max_bytes}); set VISUAL_TRUTH_UPLOAD_CMD"
                )
            blob = encode_path.read_bytes()
            b64 = base64.b64encode(blob).decode("ascii")
            video_url = "data:video/mp4;base64," + b64
            result.setdefault("upload", {"method": "base64", "bytes": size})
        else:
            raise RuntimeError(
                "validation requires VISUAL_TRUTH_ALLOW_UPLOAD=1 and either VISUAL_TRUTH_UPLOAD_CMD or VISUAL_TRUTH_ALLOW_BASE64=1"
            )

        resp = nim_chat_completion(
            base_url, api_key, model, prompt, video_url, media_type, video_fps
        )
        content = extract_message_content(resp)
        result["status"] = "completed"
        result["nim"] = {
            "base_url": base_url,
            "model": model,
            "media_type": media_type,
            "video_fps": video_fps,
        }
        result["response"] = {
            "content": content,
        }

        verdict = (content or "").strip()
        validated = False
        if re.search(r"(?im)^\s*VALIDATED\s*$", verdict):
            validated = True
        if re.search(r"(?im)^\s*STATUS\s*:\s*VALIDATED\s*$", verdict):
            validated = True
        if validated:
            result["validated"] = True
            out_path.write_text(
                json.dumps(result, indent=2, ensure_ascii=True), encoding="utf-8"
            )
            return 0

        result["validated"] = False
        status_error = re.search(r"(?im)^\s*STATUS\s*:\s*(ERROR\s*:[^\n]+)", verdict)
        if status_error:
            result["error"] = status_error.group(1).strip()[:800]
        else:
            error_line = re.search(r"(?im)^\s*(ERROR\s*:[^\n]+)", verdict)
            if error_line:
                result["error"] = error_line.group(1).strip()[:800]
            else:
                result["error"] = verdict[:800] if verdict else "empty response"
        out_path.write_text(
            json.dumps(result, indent=2, ensure_ascii=True), encoding="utf-8"
        )
        return 3 if strict else 0

    except Exception as exc:
        result["status"] = "error"
        result["error"] = str(exc)[:800]
        out_path.write_text(
            json.dumps(result, indent=2, ensure_ascii=True), encoding="utf-8"
        )
        return 2 if strict else 0


if __name__ == "__main__":
    raise SystemExit(main())
