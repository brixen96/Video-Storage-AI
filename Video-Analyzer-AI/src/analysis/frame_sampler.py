import subprocess
import os
import tempfile
from typing import List

import cv2
import numpy as np


def sample_frames(file_path: str, duration: float, n_frames: int = 20) -> List[np.ndarray]:
    """
    Extract n_frames evenly spaced from the video (avoiding first/last 5%).
    Each frame is resized to 320×180 for fast SSIM comparison.
    Returns a list of BGR numpy arrays.
    """
    frames: List[np.ndarray] = []

    if duration <= 0 or n_frames <= 0:
        return frames

    start_pct = 0.05
    end_pct = 0.95
    timestamps = [
        duration * (start_pct + (end_pct - start_pct) * i / max(n_frames - 1, 1))
        for i in range(n_frames)
    ]

    with tempfile.TemporaryDirectory() as tmpdir:
        for i, ts in enumerate(timestamps):
            out_path = os.path.join(tmpdir, f"frame_{i:03d}.jpg")
            cmd = [
                "ffmpeg",
                "-ss", f"{ts:.3f}",
                "-i", file_path,
                "-vframes", "1",
                "-vf", "scale=320:180",
                "-q:v", "3",
                "-y",
                out_path,
            ]
            try:
                subprocess.run(cmd, capture_output=True, timeout=15)
                if os.path.exists(out_path):
                    frame = cv2.imread(out_path)
                    if frame is not None:
                        frames.append(frame)
            except (subprocess.TimeoutExpired, Exception):
                continue

    return frames
