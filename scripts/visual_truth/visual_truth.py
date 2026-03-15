#!/usr/bin/env python3

import os
import signal
import subprocess
import time
from datetime import datetime, timezone
from pathlib import Path


def _utc_stamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")


def _sanitize_name(value: str) -> str:
    out = []
    for ch in value or "":
        if ch.isalnum() or ch in "._-":
            out.append(ch)
        else:
            out.append("_")
    return ("".join(out) or "step")[:120]


class VideoRecorder:
    def __init__(
        self,
        task: str,
        session: str | None = None,
        base_dir: str | None = None,
        screen_spec: str | None = None,
        fps: int | None = None,
        min_seconds: int | None = None,
        validate: bool | None = None,
        strict_validate: bool | None = None,
    ):
        self.task = task
        self.session = (
            session
            or os.environ.get("VISUAL_TRUTH_SESSION")
            or f"{_utc_stamp()}-{os.getpid()}"
        )
        self.base_dir = Path(
            base_dir or os.environ.get("VISUAL_TRUTH_DIR") or "/tmp/automation_logs"
        )
        self.screen_spec = (
            screen_spec or os.environ.get("VISUAL_TRUTH_SCREEN_SPEC") or "auto"
        )
        self.fps = int(fps or os.environ.get("VISUAL_TRUTH_FPS") or 30)
        self.min_seconds = int(
            min_seconds or os.environ.get("VISUAL_TRUTH_MIN_SECONDS") or 5
        )
        if validate is None:
            validate = os.environ.get("VISUAL_TRUTH_VALIDATE", "0").strip() == "1"
        if strict_validate is None:
            strict_validate = (
                os.environ.get("VISUAL_TRUTH_VALIDATE_STRICT", "1").strip() == "1"
            )
        self.validate = bool(validate)
        self.strict_validate = bool(strict_validate)

        self.step_dir: Path | None = None
        self.video_path: Path | None = None
        self.ffmpeg_log: Path | None = None
        self.validation_out: Path | None = None
        self._proc: subprocess.Popen | None = None
        self._started_at = 0.0

    def __enter__(self):
        self.step_dir = self.base_dir / self.session / _sanitize_name(self.task)
        self.step_dir.mkdir(parents=True, exist_ok=True)

        ts = _utc_stamp()
        self.video_path = self.step_dir / f"{ts}.mp4"
        self.ffmpeg_log = self.step_dir / f"{ts}.ffmpeg.log"
        self.validation_out = self.step_dir / f"{ts}.validation.json"

        screen_spec = self.screen_spec
        if screen_spec == "auto":
            screen_spec = "Capture screen 0"

        args = [
            "ffmpeg",
            "-nostdin",
            "-hide_banner",
            "-loglevel",
            "error",
            "-y",
            "-f",
            "avfoundation",
            "-framerate",
            str(self.fps),
            "-pixel_format",
            "nv12",
            "-i",
            screen_spec,
            "-vf",
            "format=yuv420p",
            "-c:v",
            "libx264",
            "-preset",
            "ultrafast",
            "-crf",
            "23",
            str(self.video_path),
        ]

        self._started_at = time.time()
        with open(self.ffmpeg_log, "wb") as log_fp:
            self._proc = subprocess.Popen(
                args, stdout=subprocess.DEVNULL, stderr=log_fp
            )

        ok = False
        deadline = time.time() + 4.0
        while time.time() < deadline:
            if self._proc.poll() is not None:
                break
            if self.video_path.exists() and self.video_path.stat().st_size > 4096:
                ok = True
                break
            time.sleep(0.1)

        if not ok:
            self._stop_recorder()
            raise RuntimeError(
                "Visual Truth recording failed (check macOS Screen Recording permission for your terminal app). "
                f"ffmpeg_log={self.ffmpeg_log}"
            )

        return self

    def __exit__(self, exc_type, exc, tb):
        try:
            if self._started_at and self.min_seconds > 0:
                elapsed = time.time() - self._started_at
                if elapsed < self.min_seconds:
                    time.sleep(self.min_seconds - elapsed)
        finally:
            self._stop_recorder()

        if exc_type is not None:
            return False

        if self.validate and self.video_path and self.validation_out:
            vt_validate = Path(__file__).resolve().parent / "vt_validate.py"
            env = dict(os.environ)
            env.setdefault("VISUAL_TRUTH_VALIDATE", "1")
            env.setdefault(
                "VISUAL_TRUTH_VALIDATE_STRICT", "1" if self.strict_validate else "0"
            )
            proc = subprocess.run(
                [
                    "python3",
                    str(vt_validate),
                    "--video",
                    str(self.video_path),
                    "--task",
                    self.task,
                    "--out",
                    str(self.validation_out),
                ],
                env=env,
            )
            if proc.returncode != 0 and self.strict_validate:
                raise RuntimeError(
                    f"Visual Truth validation failed rc={proc.returncode} out={self.validation_out}"
                )

        return False

    def _stop_recorder(self):
        proc = self._proc
        if not proc:
            return

        if proc.poll() is None:
            try:
                proc.send_signal(signal.SIGINT)
            except Exception:
                pass
            try:
                proc.wait(timeout=2.0)
            except Exception:
                pass

        if proc.poll() is None:
            try:
                proc.terminate()
            except Exception:
                pass
            try:
                proc.wait(timeout=1.0)
            except Exception:
                pass

        if proc.poll() is None:
            try:
                proc.kill()
            except Exception:
                pass

        self._proc = None


if __name__ == "__main__":
    with VideoRecorder(task="demo_step"):
        print("do_step()")
        time.sleep(1)
