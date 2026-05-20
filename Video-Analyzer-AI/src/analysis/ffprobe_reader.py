import subprocess
import json
import os

from src.models.video_result import VideoResult


def read_video_metadata(file_path: str) -> VideoResult:
    file_name = os.path.basename(file_path)
    file_size = os.path.getsize(file_path)
    result = VideoResult(file_path=file_path, file_name=file_name, file_size=file_size)

    try:
        cmd = [
            "ffprobe", "-v", "quiet",
            "-print_format", "json",
            "-show_format",
            "-show_streams",
            file_path,
        ]
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=60)

        if proc.returncode != 0:
            result.error = f"FFprobe error: {proc.stderr.strip() or 'unknown error'}"
            return result

        data = json.loads(proc.stdout)
        fmt = data.get("format", {})

        # Duration from format level (most reliable)
        try:
            result.duration = float(fmt.get("duration", 0))
        except (ValueError, TypeError):
            result.duration = 0.0

        video_found = False
        audio_found = False

        for stream in data.get("streams", []):
            codec_type = stream.get("codec_type")

            if codec_type == "video" and not video_found:
                video_found = True
                result.video_codec = stream.get("codec_name", "unknown")
                result.width = int(stream.get("width", 0))
                result.height = int(stream.get("height", 0))

                # Frame rate: prefer r_frame_rate (real fps)
                fps_str = stream.get("r_frame_rate", "0/1")
                try:
                    num, den = fps_str.split("/")
                    result.fps = float(num) / float(den) if float(den) else 0.0
                except Exception:
                    result.fps = 0.0

                # Stream-level bitrate, fall back to format bitrate
                raw_br = stream.get("bit_rate") or fmt.get("bit_rate", 0)
                try:
                    result.video_bitrate = int(raw_br)
                except (ValueError, TypeError):
                    result.video_bitrate = 0

                # Duration override from stream if format didn't have it
                if not result.duration:
                    try:
                        result.duration = float(stream.get("duration", 0))
                    except (ValueError, TypeError):
                        pass

            elif codec_type == "audio" and not audio_found:
                audio_found = True
                result.has_audio = True
                result.audio_codec = stream.get("codec_name", "unknown")
                result.audio_channels = int(stream.get("channels", 0))
                try:
                    result.audio_sample_rate = int(stream.get("sample_rate", 0))
                except (ValueError, TypeError):
                    result.audio_sample_rate = 0
                try:
                    result.audio_bitrate = int(stream.get("bit_rate", 0))
                except (ValueError, TypeError):
                    result.audio_bitrate = 0

        result.is_analyzed = True

    except FileNotFoundError:
        result.error = "FFprobe not found. Please install FFmpeg and ensure it is on your PATH."
    except subprocess.TimeoutExpired:
        result.error = "FFprobe timed out reading this file."
    except json.JSONDecodeError:
        result.error = "Could not parse FFprobe output."
    except Exception as e:
        result.error = str(e)

    return result
