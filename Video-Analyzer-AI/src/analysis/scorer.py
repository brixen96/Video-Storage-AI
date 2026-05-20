from typing import List

from src.models.video_result import VideoResult


# ── Codec tiers ──────────────────────────────────────────────────────────────

VIDEO_CODEC_TIERS = {
    "av1":          100,
    "hevc":          90,
    "h265":          90,
    "vp9":           80,
    "avc":           65,
    "h264":          65,
    "vp8":           50,
    "mpeg4":         40,
    "mpeg2video":    30,
    "mpeg1video":    20,
    "wmv":           25,
    "theora":        35,
}

AUDIO_CODEC_TIERS = {
    "truehd":        100,
    "flac":           95,
    "pcm_s32":        92,
    "pcm_s24":        90,
    "pcm_s16":        85,
    "eac3":           78,
    "dts":            75,
    "opus":           75,
    "aac":            70,
    "ac3":            65,
    "vorbis":         60,
    "mp3":            50,
    "mp2":            40,
    "wma":            35,
}


def _codec_score(codec: str, tiers: dict) -> float:
    codec_lower = codec.lower()
    for key, score in tiers.items():
        if key in codec_lower:
            return float(score)
    return 35.0  # unknown


def _resolution_score(width: int, height: int) -> float:
    pixels = width * height
    if pixels >= 8_000_000:   return 100.0   # 4K+
    if pixels >= 3_500_000:   return  85.0   # 2K / QHD
    if pixels >= 2_000_000:   return  70.0   # 1080p
    if pixels >= 900_000:     return  50.0   # 720p
    if pixels >= 400_000:     return  30.0   # 480p
    return 15.0


def _fps_score(fps: float) -> float:
    if fps >= 120: return 100.0
    if fps >=  60: return  90.0
    if fps >=  48: return  80.0
    if fps >=  30: return  70.0
    if fps >=  24: return  60.0
    return 40.0


def _channels_score(channels: int) -> float:
    if channels >= 8: return 100.0
    if channels >= 6: return  90.0
    if channels >= 2: return  70.0
    if channels == 1: return  40.0
    return 40.0


def _sample_rate_score(sr: int) -> float:
    if sr >= 96000: return 100.0
    if sr >= 48000: return  85.0
    if sr >= 44100: return  70.0
    return 50.0


def _relative_scores(values: List[float]) -> List[float]:
    """Normalise a list of values so the highest maps to 100."""
    max_val = max(values) if values else 0
    if max_val == 0:
        return [50.0] * len(values)
    return [round(min(100.0, v / max_val * 100), 1) if v > 0 else 20.0 for v in values]


# ── Main scoring function ─────────────────────────────────────────────────────

def calculate_scores(results: List[VideoResult]) -> None:
    """
    Populate video_score, audio_score, technical_score, and overall_score
    for every successfully analysed VideoResult in the list.
    Call this once after metadata is loaded, and again after ssim_score is set.
    """
    valid = [r for r in results if r.is_analyzed and not r.error]
    if not valid:
        return

    # Relative bitrate scoring across the group
    vbr_rel = _relative_scores([r.video_bitrate for r in valid])
    abr_rel = _relative_scores([r.audio_bitrate for r in valid])

    for i, r in enumerate(valid):

        # ── Video score ───────────────────────────────────────────────────────
        res   = _resolution_score(r.width, r.height)
        codec = _codec_score(r.video_codec, VIDEO_CODEC_TIERS)
        fps   = _fps_score(r.fps)
        vbr   = vbr_rel[i]

        if r.ssim_score is not None:
            r.video_score = round(
                res   * 0.28 +
                codec * 0.20 +
                vbr   * 0.20 +
                fps   * 0.12 +
                r.ssim_score * 0.20,
                1,
            )
        else:
            r.video_score = round(
                res   * 0.35 +
                codec * 0.25 +
                vbr   * 0.25 +
                fps   * 0.15,
                1,
            )

        # ── Audio score ───────────────────────────────────────────────────────
        if r.has_audio:
            ac  = _codec_score(r.audio_codec, AUDIO_CODEC_TIERS)
            ch  = _channels_score(r.audio_channels)
            sr  = _sample_rate_score(r.audio_sample_rate)
            abr = abr_rel[i]
            r.audio_score = round(
                ac  * 0.35 +
                ch  * 0.25 +
                sr  * 0.20 +
                abr * 0.20,
                1,
            )
        else:
            r.audio_score = 0.0

        # ── Technical score ───────────────────────────────────────────────────
        tech = 80.0

        # Container format
        ext = r.file_name.rsplit(".", 1)[-1].lower()
        if ext in ("mkv", "mp4", "mov"):
            tech += 10.0
        elif ext in ("avi", "wmv", "flv", "rm", "rmvb"):
            tech -= 10.0

        # Duration consistency (flag outliers)
        durations = [x.duration for x in valid if x.duration > 0]
        if len(durations) > 1:
            avg_dur = sum(durations) / len(durations)
            if avg_dur > 0:
                deviation = abs(r.duration - avg_dur) / avg_dur
                if deviation > 0.10:
                    tech -= 20.0
                elif deviation > 0.05:
                    tech -= 10.0

        r.technical_score = round(min(100.0, max(0.0, tech)), 1)

        # ── Overall score ─────────────────────────────────────────────────────
        r.overall_score = round(
            r.video_score    * 0.60 +
            r.audio_score    * 0.30 +
            r.technical_score * 0.10,
            1,
        )


def build_recommendation(results: List[VideoResult]) -> str:
    """Return a human-readable explanation of why the top-ranked file was chosen."""
    valid = [r for r in results if r.is_analyzed and not r.error]
    if not valid:
        return "No files could be analysed."
    if len(valid) == 1:
        return f"{valid[0].file_name} is the only successfully analysed file."

    ranked = sorted(valid, key=lambda r: r.overall_score, reverse=True)
    winner = ranked[0]
    runner = ranked[1]

    reasons: List[str] = []

    if winner.video_score > runner.video_score + 2:
        parts: List[str] = []
        if winner.width * winner.height > runner.width * runner.height:
            parts.append(f"higher resolution ({winner.resolution_str} vs {runner.resolution_str})")
        if _codec_score(winner.video_codec, VIDEO_CODEC_TIERS) > _codec_score(runner.video_codec, VIDEO_CODEC_TIERS):
            parts.append(f"better video codec ({winner.video_codec.upper()} vs {runner.video_codec.upper()})")
        if winner.video_bitrate > runner.video_bitrate * 1.1:
            parts.append(f"higher video bitrate ({winner.video_bitrate_str} vs {runner.video_bitrate_str})")
        if winner.ssim_score and runner.ssim_score and winner.ssim_score > runner.ssim_score + 2:
            parts.append(f"better perceptual quality (SSIM {winner.ssim_score:.0f} vs {runner.ssim_score:.0f})")
        if parts:
            reasons.append("Video: " + ", ".join(parts))

    if winner.audio_score > runner.audio_score + 2:
        parts = []
        if _codec_score(winner.audio_codec, AUDIO_CODEC_TIERS) > _codec_score(runner.audio_codec, AUDIO_CODEC_TIERS):
            parts.append(f"better audio codec ({winner.audio_codec.upper()} vs {runner.audio_codec.upper()})")
        if winner.audio_channels > runner.audio_channels:
            parts.append(f"more channels ({winner.audio_channels_str} vs {runner.audio_channels_str})")
        if winner.audio_bitrate > runner.audio_bitrate * 1.1:
            parts.append(f"higher audio bitrate ({winner.audio_bitrate_str} vs {runner.audio_bitrate_str})")
        if parts:
            reasons.append("Audio: " + ", ".join(parts))

    if not reasons:
        reasons.append("marginally better overall technical quality")

    return "  •  ".join(reasons)
