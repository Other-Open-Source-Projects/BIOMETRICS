#!/usr/bin/env python3
"""
Cosmos Video Generator - NVIDIA Cosmos-Transfer1-7B Integration
Enterprise-grade text-to-video generation with Qwen 3.5 VLM quality assurance

Features:
- NVIDIA Cosmos-Transfer1-7B API integration
- Text-to-Video generation (4K, 30fps support)
- Qwen 3.5 VLM quality verification
- GitLab upload for media > 1MB
- Automatic retry logic with exponential backoff
- Comprehensive progress logging
- Metadata tracking and audit trail

Best Practices 2026:
- Type hints for all functions
- Comprehensive docstrings
- Error handling with retries
- Progress logging
- Environment variable validation
"""

import requests
import os
import json
import time
import base64
import logging
from pathlib import Path
from datetime import datetime
from typing import Optional, Dict, Any, Tuple
from dotenv import load_dotenv

load_dotenv()

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


class CosmosVideoGenerator:
    """
    Enterprise video generation using NVIDIA Cosmos-Transfer1-7B

    Attributes:
        api_key: NVIDIA NIM API key
        base_url: NVIDIA NIM API endpoint
        headers: Request headers for API calls
        gitlab_token: GitLab personal access token
        gitlab_project_id: GitLab project ID for media storage
        outputs_dir: Directory for generated videos
    """

    def __init__(self):
        """Initialize Cosmos Video Generator with environment configuration"""
        self.api_key = os.getenv("NVIDIA_API_KEY")
        if not self.api_key:
            raise ValueError("NVIDIA_API_KEY not found in environment")

        self.base_url = "https://integrate.api.nvidia.com/v1"
        self.headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }

        self.gitlab_token = os.getenv("GITLAB_TOKEN")
        self.gitlab_project_id = os.getenv("GITLAB_MEDIA_PROJECT_ID")

        workspace = os.getenv("BIOMETRICS_WORKSPACE", "").strip()
        self.project_root = (
            Path(workspace).expanduser().resolve()
            if workspace
            else Path(__file__).resolve().parents[1]
        )

        self.outputs_dir = self.project_root / "outputs" / "videos"
        self.outputs_dir.mkdir(parents=True, exist_ok=True)

        logger.info("CosmosVideoGenerator initialized successfully")

    def generate_video(
        self,
        prompt: str,
        duration: int = 5,
        resolution: str = "3840x2160",
        fps: int = 30,
        model: str = "nvidia/cosmos-transfer1-7b",
    ) -> Dict[str, Any]:
        """
        Generate video from text prompt using NVIDIA Cosmos-Transfer1-7B

        Args:
            prompt: Text description of the video to generate
            duration: Video duration in seconds (default: 5)
            resolution: Video resolution (default: "3840x2160" for 4K)
            fps: Frames per second (default: 30)
            model: NVIDIA Cosmos model to use (default: cosmos-transfer1-7b)

        Returns:
            Dictionary containing:
                - video_url: URL to generated video
                - video_path: Local file path
                - metadata: Generation metadata
                - status: "success" or "error"

        Raises:
            requests.RequestException: If API call fails after retries
            ValueError: If parameters are invalid

        Example:
            >>> generator = CosmosVideoGenerator()
            >>> result = generator.generate_video(
            ...     prompt="A futuristic city with flying cars",
            ...     duration=10,
            ...     resolution="3840x2160",
            ...     fps=30
            ... )
        """
        logger.info(f"Starting video generation: {prompt[:50]}...")
        logger.info(
            f"Parameters: duration={duration}s, resolution={resolution}, fps={fps}"
        )

        max_retries = 3
        retry_delay = 5

        for attempt in range(max_retries):
            try:
                response = requests.post(
                    f"{self.base_url}/video/generations",
                    headers=self.headers,
                    json={
                        "model": model,
                        "prompt": prompt,
                        "duration": duration,
                        "resolution": resolution,
                        "fps": fps,
                    },
                    timeout=120,
                )

                response.raise_for_status()
                result = response.json()

                video_url = result.get("video_url")
                if not video_url:
                    raise ValueError("No video_url in API response")

                video_path = self._download_video(video_url, prompt)

                metadata = {
                    "prompt": prompt,
                    "duration": duration,
                    "resolution": resolution,
                    "fps": fps,
                    "model": model,
                    "generated_at": datetime.utcnow().isoformat(),
                    "video_url": video_url,
                    "video_path": str(video_path),
                }

                self.log_generation(metadata)

                logger.info(f"Video generation successful: {video_path}")
                return {
                    "status": "success",
                    "video_url": video_url,
                    "video_path": str(video_path),
                    "metadata": metadata,
                }

            except requests.RequestException as e:
                logger.warning(f"Attempt {attempt + 1}/{max_retries} failed: {str(e)}")
                if attempt < max_retries - 1:
                    time.sleep(retry_delay * (2**attempt))
                else:
                    logger.error(f"All {max_retries} attempts failed")
                    raise

        return {"status": "error", "error": "Max retries exceeded"}

    def _download_video(self, video_url: str, prompt: str) -> Path:
        """
        Download generated video to outputs directory

        Args:
            video_url: URL of the video to download
            prompt: Text prompt (used for filename)

        Returns:
            Path to downloaded video file
        """
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        safe_prompt = prompt[:30].replace(" ", "_").replace("/", "_")
        filename = f"{timestamp}_{safe_prompt}.mp4"
        video_path = self.outputs_dir / filename

        logger.info(f"Downloading video to: {video_path}")

        response = requests.get(video_url, stream=True)
        response.raise_for_status()

        with open(video_path, "wb") as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)

        file_size = video_path.stat().st_size
        logger.info(f"Download complete: {file_size / (1024 * 1024):.2f} MB")

        return video_path

    def check_quality(self, video_path: str) -> bool:
        """
        Verify video quality using Qwen 3.5 VLM

        Checks for:
        - Physical correctness (gravity, light, shadows)
        - No artifacts or glitches
        - Consistent lighting
        - Brand identity preservation

        Args:
            video_path: Path to video file to verify

        Returns:
            True if video passes quality check, False otherwise

        Example:
            >>> generator = CosmosVideoGenerator()
            >>> is_approved = generator.check_quality("output.mp4")
            >>> if is_approved:
            ...     print("APPROVED FOR PRODUCTION")
        """
        logger.info(f"Starting quality check: {video_path}")

        try:
            with open(video_path, "rb") as f:
                video_base64 = base64.b64encode(f.read()).decode("utf-8")

            prompt = """
Prüfe dieses Video auf:
1. Physikalische Korrektheit (Schwerkraft, Licht, Schatten)
2. Keine Artefakte/Glitches
3. Konsistente Beleuchtung
4. Marken-Identität gewahrt?

Wenn FEHLER gefunden:
→ Liste ALLE Fehler auf
→ Empfehle Korrektur mit cosmos-video-edit

Wenn PERFEKT:
→ Bestätige "APPROVED FOR PRODUCTION"
"""

            response = requests.post(
                "https://integrate.api.nvidia.com/v1/chat/completions",
                headers={
                    "Authorization": f"Bearer {self.api_key}",
                    "Content-Type": "application/json",
                },
                json={
                    "model": "qwen/qwen3.5-397b-a17b",
                    "messages": [
                        {
                            "role": "user",
                            "content": [
                                {"type": "text", "text": prompt},
                                {
                                    "type": "file",
                                    "url": f"data:video/mp4;base64,{video_base64}",
                                },
                            ],
                        }
                    ],
                },
                timeout=120,
            )

            response.raise_for_status()
            result = response.json()

            analysis = result["choices"][0]["message"]["content"]
            logger.info(f"Quality analysis: {analysis[:200]}...")

            is_approved = "APPROVED FOR PRODUCTION" in analysis.upper()

            if is_approved:
                logger.info("✅ Video APPROVED FOR PRODUCTION")
            else:
                logger.warning("⚠️ Video requires corrections")
                logger.warning(analysis)

            return is_approved

        except Exception as e:
            logger.error(f"Quality check failed: {str(e)}")
            return False

    def upload_to_gitlab(self, video_path: str) -> str:
        """
        Upload video to GitLab media repository

        Required for files > 1MB per project mandates

        Args:
            video_path: Path to video file to upload

        Returns:
            Public URL of uploaded video

        Raises:
            requests.RequestException: If upload fails
            ValueError: If GitLab credentials not configured

        Example:
            >>> generator = CosmosVideoGenerator()
            >>> url = generator.upload_to_gitlab("output.mp4")
            >>> print(f"Video available at: {url}")
        """
        logger.info(f"Uploading to GitLab: {video_path}")

        if not self.gitlab_token or not self.gitlab_project_id:
            raise ValueError("GITLAB_TOKEN or GITLAB_MEDIA_PROJECT_ID not configured")

        file_size = Path(video_path).stat().st_size
        if file_size > 1024 * 1024:
            logger.info(
                f"File size: {file_size / (1024 * 1024):.2f} MB (> 1MB, GitLab required)"
            )

        with open(video_path, "rb") as f:
            response = requests.post(
                f"https://gitlab.com/api/v4/projects/{self.gitlab_project_id}/uploads",
                headers={"PRIVATE-TOKEN": self.gitlab_token},
                files={"file": f},
            )

        response.raise_for_status()
        result = response.json()

        public_url = result.get("full_path")
        if not public_url:
            raise ValueError("No full_path in GitLab response")

        full_url = f"https://gitlab.com{public_url}"

        logger.info(f"✅ Uploaded to GitLab: {full_url}")

        return full_url

    def save_to_outputs(self, video_path: str) -> Dict[str, Any]:
        """
        Save video metadata to outputs directory

        Creates:
        - Video file in outputs/videos/
        - Metadata JSON in outputs/metadata/

        Args:
            video_path: Path to generated video

        Returns:
            Dictionary with save locations and status
        """
        metadata_dir = self.project_root / "outputs" / "metadata"
        metadata_dir.mkdir(parents=True, exist_ok=True)

        video_file = Path(video_path)
        metadata_file = metadata_dir / f"{video_file.stem}.json"

        metadata = {
            "video_path": str(video_path),
            "filename": video_file.name,
            "size_bytes": video_file.stat().st_size,
            "created_at": datetime.utcnow().isoformat(),
            "status": "generated",
        }

        with open(metadata_file, "w") as f:
            json.dump(metadata, f, indent=2)

        logger.info(f"Metadata saved: {metadata_file}")

        return {
            "video_path": str(video_path),
            "metadata_path": str(metadata_file),
            "status": "saved",
        }

    def log_generation(self, metadata: Dict[str, Any]) -> None:
        """
        Log generation metadata for audit trail

        Appends to logs/generation_log.jsonl (JSON Lines format)

        Args:
            metadata: Generation metadata dictionary

        Example:
            {
                "prompt": "futuristic city",
                "duration": 10,
                "resolution": "3840x2160",
                "fps": 30,
                "model": "nvidia/cosmos-transfer1-7b",
                "generated_at": "2026-02-19T18:42:00",
                "video_url": "https://...",
                "video_path": "/path/to/video.mp4"
            }
        """
        log_dir = self.project_root / "logs"
        log_dir.mkdir(parents=True, exist_ok=True)

        log_file = log_dir / "generation_log.jsonl"

        log_entry = {"timestamp": datetime.utcnow().isoformat(), "metadata": metadata}

        with open(log_file, "a") as f:
            f.write(json.dumps(log_entry) + "\n")

        logger.info(f"Generation logged: {log_file}")


def main():
    """Main entry point for CLI usage"""
    import argparse

    parser = argparse.ArgumentParser(
        description="Generate videos with NVIDIA Cosmos-Transfer1-7B",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --prompt "A futuristic city with flying cars"
  %(prog)s --prompt "Ocean waves at sunset" --duration 10 --fps 60
  %(prog)s --prompt "Product showcase" --resolution "1920x1080"
        """,
    )

    parser.add_argument(
        "--prompt", "-p", required=True, help="Text prompt for video generation"
    )
    parser.add_argument(
        "--duration",
        "-d",
        type=int,
        default=5,
        help="Video duration in seconds (default: 5)",
    )
    parser.add_argument(
        "--resolution",
        "-r",
        default="3840x2160",
        help="Video resolution (default: 3840x2160 for 4K)",
    )
    parser.add_argument(
        "--fps", type=int, default=30, help="Frames per second (default: 30)"
    )
    parser.add_argument(
        "--model",
        "-m",
        default="nvidia/cosmos-transfer1-7b",
        help="NVIDIA Cosmos model (default: nvidia/cosmos-transfer1-7b)",
    )
    parser.add_argument(
        "--skip-quality-check",
        action="store_true",
        help="Skip Qwen 3.5 VLM quality verification",
    )
    parser.add_argument("--skip-gitlab", action="store_true", help="Skip GitLab upload")

    args = parser.parse_args()

    try:
        generator = CosmosVideoGenerator()

        logger.info("=" * 60)
        logger.info("🎬 COSMOS VIDEO GENERATION STARTED")
        logger.info("=" * 60)

        result = generator.generate_video(
            prompt=args.prompt,
            duration=args.duration,
            resolution=args.resolution,
            fps=args.fps,
            model=args.model,
        )

        if result["status"] != "success":
            logger.error("❌ Video generation failed")
            return 1

        video_path = result["video_path"]
        logger.info(f"✅ Video generated: {video_path}")

        generator.save_to_outputs(video_path)

        if not args.skip_quality_check:
            logger.info("🔍 Starting quality check...")
            is_approved = generator.check_quality(video_path)

            if not is_approved:
                logger.warning("⚠️ Video did not pass quality check")
                logger.warning("Manual review recommended before production use")

        if not args.skip_gitlab:
            logger.info("📤 Uploading to GitLab...")
            gitlab_url = generator.upload_to_gitlab(video_path)
            logger.info(f"✅ GitLab URL: {gitlab_url}")

        logger.info("=" * 60)
        logger.info("🎉 VIDEO GENERATION COMPLETE")
        logger.info("=" * 60)
        logger.info(f"Video: {video_path}")
        if not args.skip_gitlab:
            logger.info(f"GitLab: {gitlab_url}")
        logger.info("=" * 60)

        return 0

    except Exception as e:
        logger.error(f"❌ Generation failed: {str(e)}")
        return 1


if __name__ == "__main__":
    exit(main())
