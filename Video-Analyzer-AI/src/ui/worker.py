import os
from typing import List

from PyQt6.QtCore import QThread, pyqtSignal

from src.analysis.ffprobe_reader import read_video_metadata
from src.analysis.frame_sampler import sample_frames
from src.analysis.ssim_scorer import compute_ssim_pair
from src.analysis.scorer import calculate_scores
from src.models.video_result import VideoResult


class AnalysisWorker(QThread):
    # (file_name, percent 0–100, status_text)
    progress = pyqtSignal(str, int, str)
    all_done  = pyqtSignal(list)   # List[VideoResult]

    def __init__(self, file_paths: List[str], parent=None):
        super().__init__(parent)
        self.file_paths = file_paths
        self._abort = False

    def abort(self):
        self._abort = True

    def run(self):
        results: List[VideoResult] = []

        # ── Phase 1: Metadata ─────────────────────────────────────────────────
        for path in self.file_paths:
            if self._abort:
                return
            name = os.path.basename(path)
            self.progress.emit(name, 5, "Reading metadata…")
            result = read_video_metadata(path)
            if result.error:
                self.progress.emit(name, 100, f"Error: {result.error}")
            else:
                self.progress.emit(name, 30, "Metadata read")
            results.append(result)

        # ── Phase 2: Initial scores (metadata only) ────────────────────────
        calculate_scores(results)

        # ── Phase 3: Frame sampling ───────────────────────────────────────────
        valid = [r for r in results if r.is_analyzed and not r.error]
        frames_map: dict = {}

        if len(valid) >= 2:
            for r in valid:
                if self._abort:
                    return
                self.progress.emit(r.file_name, 40, "Sampling frames…")
                frames = sample_frames(r.file_path, r.duration, n_frames=20)
                frames_map[r.file_path] = frames
                self.progress.emit(r.file_name, 70, f"Sampled {len(frames)} frames")

            # ── Phase 4: SSIM comparison ──────────────────────────────────────
            # Use the highest-metadata-scored file as the reference
            reference = max(valid, key=lambda r: r.video_score)
            ref_frames = frames_map.get(reference.file_path, [])

            for r in valid:
                if self._abort:
                    return
                if r.file_path == reference.file_path:
                    r.ssim_score = 100.0
                else:
                    other_frames = frames_map.get(r.file_path, [])
                    self.progress.emit(r.file_name, 80, "Computing SSIM…")
                    r.ssim_score = compute_ssim_pair(ref_frames, other_frames)

            # ── Phase 5: Recalculate scores with SSIM ──────────────────────
            calculate_scores(results)

        # Emit final progress
        for r in results:
            if not r.error:
                self.progress.emit(r.file_name, 100, "Complete")

        self.all_done.emit(results)
