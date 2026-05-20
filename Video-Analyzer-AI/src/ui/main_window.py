from typing import Dict, List, Optional

from PyQt6.QtWidgets import (
    QMainWindow, QWidget, QVBoxLayout, QHBoxLayout,
    QPushButton, QLabel, QProgressBar, QFrame, QScrollArea,
    QSizePolicy,
)
from PyQt6.QtCore import Qt
from PyQt6.QtGui import QFont

from src.ui.drop_zone import DropZone
from src.ui.results_panel import ResultsPanel
from src.ui.worker import AnalysisWorker
from src.models.video_result import VideoResult


STYLESHEET = """
/* ── Base ──────────────────────────────────────────────────────────────── */
QMainWindow, QWidget {
    background-color: #0f0f1a;
    color: #e0e0f0;
    font-family: 'Segoe UI', Arial, sans-serif;
    font-size: 13px;
}

QScrollArea, QScrollArea > QWidget > QWidget {
    background-color: transparent;
}

QScrollBar:vertical {
    background: #1a1a2e;
    width: 8px;
    border-radius: 4px;
}
QScrollBar::handle:vertical {
    background: #3a3a5c;
    border-radius: 4px;
    min-height: 30px;
}
QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical { height: 0; }

/* ── Drop zone ──────────────────────────────────────────────────────────── */
QFrame#dropFrame {
    border: 2px dashed #3a3a5c;
    border-radius: 10px;
    background-color: #1a1a2e;
}
QFrame#dropFrame[dragOver="true"] {
    border-color: #6366f1;
    background-color: #1e1e3a;
}

QLabel#dropHint {
    color: #6060a0;
    font-size: 14px;
}

QLabel#fileRowName {
    color: #c0c0e0;
}
QLabel#fileRowSize {
    color: #606080;
    font-size: 12px;
}

QPushButton#removeBtn {
    background: transparent;
    color: #505070;
    border: none;
    font-size: 12px;
    border-radius: 4px;
}
QPushButton#removeBtn:hover {
    background: #2a1a2e;
    color: #ef4444;
}

/* ── Buttons ────────────────────────────────────────────────────────────── */
QPushButton#browseBtn {
    background-color: #252540;
    color: #a0a0d0;
    border: 1px solid #3a3a5c;
    border-radius: 6px;
    padding: 6px 16px;
}
QPushButton#browseBtn:hover {
    background-color: #2e2e50;
    border-color: #6366f1;
    color: #e0e0f0;
}

QPushButton#clearBtn {
    background-color: transparent;
    color: #606080;
    border: 1px solid #2a2a40;
    border-radius: 6px;
    padding: 6px 16px;
}
QPushButton#clearBtn:hover {
    border-color: #ef4444;
    color: #ef4444;
}

QPushButton#analyzeBtn {
    background-color: #6366f1;
    color: #ffffff;
    border: none;
    border-radius: 6px;
    padding: 8px 28px;
    font-size: 14px;
    font-weight: bold;
}
QPushButton#analyzeBtn:hover {
    background-color: #7c7ffc;
}
QPushButton#analyzeBtn:disabled {
    background-color: #252540;
    color: #404060;
}

/* ── Progress ───────────────────────────────────────────────────────────── */
QLabel#progressLabel {
    color: #8080b0;
    font-size: 12px;
}
QProgressBar {
    background-color: #1a1a2e;
    border: none;
    border-radius: 4px;
    height: 6px;
    text-visible: false;
}
QProgressBar::chunk {
    background-color: #6366f1;
    border-radius: 4px;
}

/* ── Results table ──────────────────────────────────────────────────────── */
QLabel#sectionTitle {
    color: #8080b0;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1px;
    padding-bottom: 4px;
}

QTableWidget#resultsTable {
    background-color: #1a1a2e;
    alternate-background-color: #161626;
    border: none;
    border-radius: 8px;
    gridline-color: transparent;
    selection-background-color: #2a2a4a;
    selection-color: #e0e0f0;
    outline: none;
}
QTableWidget#resultsTable::item {
    padding: 0 8px;
    border: none;
}
QHeaderView::section {
    background-color: #141428;
    color: #6060a0;
    font-size: 11px;
    font-weight: bold;
    border: none;
    padding: 6px 8px;
    text-transform: uppercase;
}
QHeaderView::section:first {
    border-top-left-radius: 8px;
}

/* ── Recommendation banner ──────────────────────────────────────────────── */
QFrame#recommendationBanner {
    background-color: #1a2a1e;
    border: 1px solid #22c55e44;
    border-radius: 8px;
}
QLabel#bannerIcon {
    color: #fbbf24;
    font-size: 16px;
}
QLabel#bannerText {
    color: #c0d0c0;
    font-size: 13px;
    line-height: 1.5;
}

/* ── Divider ────────────────────────────────────────────────────────────── */
QFrame#divider {
    background-color: #2a2a40;
    max-height: 1px;
}
"""


class _ProgressRow(QWidget):
    def __init__(self, file_name: str, parent=None):
        super().__init__(parent)
        layout = QHBoxLayout(self)
        layout.setContentsMargins(0, 2, 0, 2)
        layout.setSpacing(10)

        self._name = QLabel(file_name)
        self._name.setObjectName("progressLabel")
        self._name.setFixedWidth(220)
        self._name.setToolTip(file_name)
        layout.addWidget(self._name)

        self._bar = QProgressBar()
        self._bar.setRange(0, 100)
        self._bar.setValue(0)
        self._bar.setFixedHeight(6)
        self._bar.setTextVisible(False)
        layout.addWidget(self._bar, stretch=1)

        self._status = QLabel("Waiting…")
        self._status.setObjectName("progressLabel")
        self._status.setFixedWidth(160)
        layout.addWidget(self._status)

    def update(self, percent: int, status: str):
        self._bar.setValue(percent)
        self._status.setText(status)


class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("Video Analyzer AI")
        self.setMinimumSize(900, 660)
        self.resize(1050, 740)

        self._worker: Optional[AnalysisWorker] = None
        self._progress_rows: Dict[str, _ProgressRow] = {}

        self._build_ui()
        self.setStyleSheet(STYLESHEET)

    # ── UI construction ───────────────────────────────────────────────────────

    def _build_ui(self):
        central = QWidget()
        self.setCentralWidget(central)
        root = QVBoxLayout(central)
        root.setContentsMargins(20, 20, 20, 20)
        root.setSpacing(14)

        # Title
        title = QLabel("Video Analyzer AI")
        title_font = QFont("Segoe UI", 18, QFont.Weight.Bold)
        title.setFont(title_font)
        title.setStyleSheet("color: #e0e0f0;")
        root.addWidget(title)

        subtitle = QLabel("Compare multiple video files and find the best quality version")
        subtitle.setStyleSheet("color: #606080; font-size: 12px;")
        root.addWidget(subtitle)

        divider = QFrame()
        divider.setObjectName("divider")
        divider.setFrameShape(QFrame.Shape.HLine)
        root.addWidget(divider)

        # Drop zone
        self._drop_zone = DropZone()
        self._drop_zone.files_changed.connect(self._on_files_changed)
        root.addWidget(self._drop_zone)

        # Button row
        btn_row = QHBoxLayout()
        btn_row.setSpacing(8)

        self._browse_btn = QPushButton("Browse Files")
        self._browse_btn.setObjectName("browseBtn")
        self._browse_btn.setCursor(Qt.CursorShape.PointingHandCursor)
        self._browse_btn.clicked.connect(self._drop_zone.browse)
        btn_row.addWidget(self._browse_btn)

        self._clear_btn = QPushButton("Clear All")
        self._clear_btn.setObjectName("clearBtn")
        self._clear_btn.setCursor(Qt.CursorShape.PointingHandCursor)
        self._clear_btn.clicked.connect(self._on_clear)
        self._clear_btn.setEnabled(False)
        btn_row.addWidget(self._clear_btn)

        btn_row.addStretch()

        self._analyze_btn = QPushButton("▶  Analyze")
        self._analyze_btn.setObjectName("analyzeBtn")
        self._analyze_btn.setCursor(Qt.CursorShape.PointingHandCursor)
        self._analyze_btn.clicked.connect(self._on_analyze)
        self._analyze_btn.setEnabled(False)
        btn_row.addWidget(self._analyze_btn)

        root.addLayout(btn_row)

        # Progress section
        self._progress_section = QWidget()
        prog_layout = QVBoxLayout(self._progress_section)
        prog_layout.setContentsMargins(0, 0, 0, 0)
        prog_layout.setSpacing(4)

        prog_title = QLabel("ANALYSIS PROGRESS")
        prog_title.setObjectName("sectionTitle")
        prog_layout.addWidget(prog_title)

        prog_scroll = QScrollArea()
        prog_scroll.setWidgetResizable(True)
        prog_scroll.setFrameShape(QFrame.Shape.NoFrame)
        prog_scroll.setMaximumHeight(130)

        self._progress_container = QWidget()
        self._progress_layout = QVBoxLayout(self._progress_container)
        self._progress_layout.setContentsMargins(0, 0, 0, 0)
        self._progress_layout.setSpacing(2)
        self._progress_layout.addStretch()

        prog_scroll.setWidget(self._progress_container)
        prog_layout.addWidget(prog_scroll)

        self._progress_section.setVisible(False)
        root.addWidget(self._progress_section)

        # Results panel
        self._results = ResultsPanel()
        root.addWidget(self._results, stretch=1)

    # ── Slots ─────────────────────────────────────────────────────────────────

    def _on_files_changed(self, paths: List[str]):
        has_files = bool(paths)
        self._analyze_btn.setEnabled(has_files and len(paths) >= 2)
        self._clear_btn.setEnabled(has_files)
        if len(paths) == 1:
            self._analyze_btn.setToolTip("Add at least 2 files to compare")
        else:
            self._analyze_btn.setToolTip("")

    def _on_clear(self):
        if self._worker and self._worker.isRunning():
            self._worker.abort()
            self._worker.wait()
        self._drop_zone.clear()
        self._results.clear()
        self._progress_section.setVisible(False)
        self._clear_progress_rows()

    def _on_analyze(self):
        paths = self._drop_zone.file_paths()
        if len(paths) < 2:
            return

        # Reset state
        self._results.clear()
        self._clear_progress_rows()
        self._progress_rows = {}
        self._progress_section.setVisible(True)

        # Build progress rows
        for p in paths:
            import os
            name = os.path.basename(p)
            row = _ProgressRow(name)
            self._progress_rows[name] = row
            self._progress_layout.insertWidget(self._progress_layout.count() - 1, row)

        # Lock controls
        self._analyze_btn.setEnabled(False)
        self._analyze_btn.setText("⏳  Analyzing…")
        self._browse_btn.setEnabled(False)
        self._clear_btn.setEnabled(False)

        # Start worker
        self._worker = AnalysisWorker(paths)
        self._worker.progress.connect(self._on_progress)
        self._worker.all_done.connect(self._on_done)
        self._worker.start()

    def _on_progress(self, file_name: str, percent: int, status: str):
        row = self._progress_rows.get(file_name)
        if row:
            row.update(percent, status)

    def _on_done(self, results: List[VideoResult]):
        self._results.show_results(results)
        self._analyze_btn.setEnabled(True)
        self._analyze_btn.setText("▶  Analyze")
        self._browse_btn.setEnabled(True)
        self._clear_btn.setEnabled(True)

    # ── Helpers ───────────────────────────────────────────────────────────────

    def _clear_progress_rows(self):
        while self._progress_layout.count() > 1:
            item = self._progress_layout.takeAt(0)
            if item.widget():
                item.widget().deleteLater()
        self._progress_rows = {}

    def closeEvent(self, event):
        if self._worker and self._worker.isRunning():
            self._worker.abort()
            self._worker.wait()
        event.accept()
