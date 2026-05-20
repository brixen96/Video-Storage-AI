from typing import List

from PyQt6.QtWidgets import (
    QWidget, QVBoxLayout, QHBoxLayout, QLabel,
    QTableWidget, QTableWidgetItem, QHeaderView, QFrame,
)
from PyQt6.QtCore import Qt
from PyQt6.QtGui import QColor, QFont

from src.models.video_result import VideoResult
from src.analysis.scorer import build_recommendation


# ── Colour helpers ────────────────────────────────────────────────────────────

def _score_color(score: float) -> QColor:
    if score >= 80:
        return QColor("#22c55e")   # green
    if score >= 60:
        return QColor("#f59e0b")   # amber
    return QColor("#ef4444")       # red


_COLUMNS = [
    ("File",       "file_name",         200),
    ("Resolution", "resolution_str",    130),
    ("V.Codec",    "video_codec",        80),
    ("FPS",        "fps_str",            55),
    ("V.Bitrate",  "video_bitrate_str",  90),
    ("A.Codec",    "audio_codec",        80),
    ("Channels",   "audio_channels_str", 70),
    ("Video",      "_video_score",       60),
    ("Audio",      "_audio_score",       60),
    ("Tech",       "_technical_score",   55),
    ("Overall",    "_overall_score",     70),
]


class ResultsPanel(QWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        self._build_ui()

    def _build_ui(self):
        layout = QVBoxLayout(self)
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(8)

        # Title row
        title = QLabel("Results")
        title.setObjectName("sectionTitle")
        layout.addWidget(title)

        # Table
        self._table = QTableWidget(0, len(_COLUMNS))
        self._table.setObjectName("resultsTable")
        self._table.setHorizontalHeaderLabels([c[0] for c in _COLUMNS])
        self._table.verticalHeader().setVisible(False)
        self._table.setEditTriggers(QTableWidget.EditTrigger.NoEditTriggers)
        self._table.setSelectionBehavior(QTableWidget.SelectionBehavior.SelectRows)
        self._table.setAlternatingRowColors(True)
        self._table.setShowGrid(False)

        hh = self._table.horizontalHeader()
        for col, (_, _, width) in enumerate(_COLUMNS):
            self._table.setColumnWidth(col, width)
        hh.setSectionResizeMode(0, QHeaderView.ResizeMode.Stretch)

        layout.addWidget(self._table)

        # Recommendation banner
        self._banner = QFrame()
        self._banner.setObjectName("recommendationBanner")
        banner_layout = QHBoxLayout(self._banner)
        banner_layout.setContentsMargins(16, 10, 16, 10)

        self._banner_icon = QLabel("★")
        self._banner_icon.setObjectName("bannerIcon")
        self._banner_icon.setFixedWidth(20)
        banner_layout.addWidget(self._banner_icon)

        self._banner_text = QLabel()
        self._banner_text.setObjectName("bannerText")
        self._banner_text.setWordWrap(True)
        banner_layout.addWidget(self._banner_text, stretch=1)

        self._banner.setVisible(False)
        layout.addWidget(self._banner)

    # ── Public API ────────────────────────────────────────────────────────────

    def clear(self):
        self._table.setRowCount(0)
        self._banner.setVisible(False)

    def show_results(self, results: List[VideoResult]):
        self.clear()

        valid = [r for r in results if r.is_analyzed and not r.error]
        ranked = sorted(valid, key=lambda r: r.overall_score, reverse=True)
        errors = [r for r in results if r.error]

        all_rows = ranked + errors
        if not all_rows:
            return

        winner_path = ranked[0].file_path if ranked else None
        self._table.setRowCount(len(all_rows))

        bold_font = QFont()
        bold_font.setBold(True)

        for row_idx, result in enumerate(all_rows):
            is_winner = (result.file_path == winner_path and not result.error)
            values = self._extract_values(result)

            for col_idx, value in enumerate(values):
                item = QTableWidgetItem(value)
                item.setTextAlignment(Qt.AlignmentFlag.AlignVCenter | Qt.AlignmentFlag.AlignLeft)

                # Colour score columns
                col_key = _COLUMNS[col_idx][1]
                if col_key in ("_video_score", "_audio_score", "_technical_score", "_overall_score"):
                    try:
                        score_val = float(value)
                        item.setForeground(_score_color(score_val))
                    except ValueError:
                        pass
                    item.setTextAlignment(Qt.AlignmentFlag.AlignVCenter | Qt.AlignmentFlag.AlignHCenter)

                if is_winner:
                    item.setFont(bold_font)
                    if col_key == "file_name":
                        item.setText(f"★  {value}")
                        item.setForeground(QColor("#fbbf24"))  # gold star

                if result.error:
                    item.setForeground(QColor("#ef4444"))
                    if col_key == "file_name":
                        item.setText(f"✕  {value}")

                self._table.setItem(row_idx, col_idx, item)

            self._table.setRowHeight(row_idx, 32)

        # Recommendation banner
        if len(ranked) >= 2:
            winner = ranked[0]
            reason = build_recommendation(results)
            self._banner_text.setText(
                f"<b>Recommended: {winner.file_name}</b><br>"
                f"<span style='color:#a0a0c0'>{reason}</span>"
            )
            self._banner.setVisible(True)
        elif len(ranked) == 1:
            self._banner_text.setText(f"<b>{ranked[0].file_name}</b> — only successfully analysed file.")
            self._banner.setVisible(True)

    # ── Helpers ───────────────────────────────────────────────────────────────

    def _extract_values(self, r: VideoResult) -> List[str]:
        if r.error:
            return [r.file_name] + ["—"] * (len(_COLUMNS) - 1)

        return [
            r.file_name,
            r.resolution_str,
            r.video_codec.upper() if r.video_codec != "unknown" else "—",
            r.fps_str,
            r.video_bitrate_str,
            r.audio_codec.upper() if r.has_audio else "—",
            r.audio_channels_str if r.has_audio else "—",
            str(r.video_score),
            str(r.audio_score) if r.has_audio else "—",
            str(r.technical_score),
            str(r.overall_score),
        ]
