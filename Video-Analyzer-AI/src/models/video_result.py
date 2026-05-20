from dataclasses import dataclass, field
from typing import Optional


@dataclass
class VideoResult:
    file_path: str
    file_name: str
    file_size: int  # bytes

    # Video stream
    video_codec: str = "unknown"
    width: int = 0
    height: int = 0
    fps: float = 0.0
    duration: float = 0.0  # seconds
    video_bitrate: int = 0  # bps

    # Audio stream
    audio_codec: str = "unknown"
    audio_channels: int = 0
    audio_sample_rate: int = 0
    audio_bitrate: int = 0  # bps
    has_audio: bool = False

    # Scores (0–100)
    video_score: float = 0.0
    audio_score: float = 0.0
    technical_score: float = 0.0
    overall_score: float = 0.0

    # Perceptual quality via SSIM (set during cross-comparison, 0–100)
    ssim_score: Optional[float] = None

    # Status
    error: Optional[str] = None
    is_analyzed: bool = False

    @property
    def resolution_str(self) -> str:
        if self.width and self.height:
            pixels = self.width * self.height
            if pixels >= 8_000_000:
                label = "4K"
            elif pixels >= 3_500_000:
                label = "2K"
            elif pixels >= 2_000_000:
                label = "1080p"
            elif pixels >= 900_000:
                label = "720p"
            elif pixels >= 400_000:
                label = "480p"
            else:
                label = "SD"
            return f"{self.width}×{self.height} ({label})"
        return "Unknown"

    @property
    def file_size_str(self) -> str:
        size = float(self.file_size)
        for unit in ("B", "KB", "MB", "GB"):
            if size < 1024:
                return f"{size:.1f} {unit}"
            size /= 1024
        return f"{size:.1f} TB"

    @property
    def duration_str(self) -> str:
        if not self.duration:
            return "Unknown"
        h = int(self.duration // 3600)
        m = int((self.duration % 3600) // 60)
        s = int(self.duration % 60)
        if h:
            return f"{h}:{m:02d}:{s:02d}"
        return f"{m}:{s:02d}"

    @property
    def fps_str(self) -> str:
        if not self.fps:
            return "Unknown"
        return f"{self.fps:.3f}".rstrip("0").rstrip(".")

    @property
    def video_bitrate_str(self) -> str:
        if not self.video_bitrate:
            return "Unknown"
        if self.video_bitrate >= 1_000_000:
            return f"{self.video_bitrate / 1_000_000:.2f} Mbps"
        return f"{self.video_bitrate / 1000:.0f} Kbps"

    @property
    def audio_bitrate_str(self) -> str:
        if not self.audio_bitrate:
            return "Unknown"
        if self.audio_bitrate >= 1_000_000:
            return f"{self.audio_bitrate / 1_000_000:.2f} Mbps"
        return f"{self.audio_bitrate / 1000:.0f} Kbps"

    @property
    def audio_channels_str(self) -> str:
        mapping = {1: "Mono", 2: "Stereo", 6: "5.1", 8: "7.1"}
        return mapping.get(self.audio_channels, str(self.audio_channels) if self.audio_channels else "—")
