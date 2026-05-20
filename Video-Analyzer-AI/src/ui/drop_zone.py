import os
from typing import List

from PyQt6.QtWidgets import (
    QWidget, QVBoxLayout, QHBoxLayout, QLabel,
    QPushButton, QFileDialog, QFrame, QScrollArea,
)
from PyQt6.QtCore import Qt, pyqtSignal
from PyQt6.QtGui import QDragEnterEvent, QDropEvent


VIDEO_EXTENSIONS = {
    ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv",
    ".m4v", ".mpg", ".mpeg", ".ts", ".webm", ".rmvb", ".rm",
}


class _FileRow(QWidget):
    removed = pyqtSignal(str)  # emits file_path

    def __init__(self, file_path: str, parent=None):
        super().__init__(parent)
        self.file_path = file_path
        layout = QHBoxLayout(self)
        layout.setContentsMargins(8, 4, 8, 4)
        layout.setSpacing(10)

        icon = QLabel("🎬")
        icon.setFixedWidth(22)
        layout.addWidget(icon)

        name_label = QLabel(os.path.basename(file_path))
        name_label.setObjectName("fileRowName")
        name_label.setToolTip(file_path)
        layout.addWidget(name_label, stretch=1)

        size = os.path.getsize(file_path)
        size_label = QLabel(_fmt_size(size))
        size_label.setObjectName("fileRowSize")
        size_label.setFixedWidth(72)
        size_label.setAlignment(Qt.AlignmentFlag.AlignRight | Qt.AlignmentFlag.AlignVCenter)
        layout.addWidget(size_label)

        remove_btn = QPushButton("✕")
        remove_btn.setObjectName("removeBtn")
        remove_btn.setFixedSize(22, 22)
        remove_btn.setCursor(Qt.CursorShape.PointingHandCursor)
        remove_btn.clicked.connect(lambda: self.removed.emit(self.file_path))
        layout.addWidget(remove_btn)


def _fmt_size(size: int) -> str:
    for unit in ("B", "KB", "MB", "GB"):
        if size < 1024:
            return f"{size:.1f} {unit}"
        size /= 1024  # type: ignore[assignment]
    return f"{size:.1f} TB"


class DropZone(QWidget):
    files_changed = pyqtSignal(list)  # emits List[str] of current file paths

    def __init__(self, parent=None):
        super().__init__(parent)
        self._paths: List[str] = []
        self._build_ui()
        self.setAcceptDrops(True)

    # ── Public ────────────────────────────────────────────────────────────────

    def file_paths(self) -> List[str]:
        return list(self._paths)

    def clear(self):
        self._paths.clear()
        self._refresh_list()
        self.files_changed.emit(self._paths)

    # ── UI construction ───────────────────────────────────────────────────────

    def _build_ui(self):
        outer = QVBoxLayout(self)
        outer.setContentsMargins(0, 0, 0, 0)
        outer.setSpacing(0)

        # Drop target frame
        self._drop_frame = QFrame()
        self._drop_frame.setObjectName("dropFrame")
        self._drop_frame.setMinimumHeight(90)

        drop_layout = QVBoxLayout(self._drop_frame)
        drop_layout.setAlignment(Qt.AlignmentFlag.AlignCenter)

        self._hint_label = QLabel("Drop video files here  or  Browse")
        self._hint_label.setObjectName("dropHint")
        self._hint_label.setAlignment(Qt.AlignmentFlag.AlignCenter)
        drop_layout.addWidget(self._hint_label)

        outer.addWidget(self._drop_frame)

        # Scroll area for file list
        self._scroll = QScrollArea()
        self._scroll.setWidgetResizable(True)
        self._scroll.setFrameShape(QFrame.Shape.NoFrame)
        self._scroll.setMaximumHeight(180)
        self._scroll.setVisible(False)

        self._list_container = QWidget()
        self._list_layout = QVBoxLayout(self._list_container)
        self._list_layout.setContentsMargins(0, 4, 0, 0)
        self._list_layout.setSpacing(2)
        self._list_layout.addStretch()

        self._scroll.setWidget(self._list_container)
        outer.addWidget(self._scroll)

    # ── Drag & drop ───────────────────────────────────────────────────────────

    def dragEnterEvent(self, event: QDragEnterEvent):
        if event.mimeData().hasUrls():
            event.acceptProposedAction()
            self._drop_frame.setProperty("dragOver", True)
            self._drop_frame.setStyle(self._drop_frame.style())

    def dragLeaveEvent(self, event):
        self._drop_frame.setProperty("dragOver", False)
        self._drop_frame.setStyle(self._drop_frame.style())

    def dropEvent(self, event: QDropEvent):
        self._drop_frame.setProperty("dragOver", False)
        self._drop_frame.setStyle(self._drop_frame.style())
        paths = [url.toLocalFile() for url in event.mimeData().urls()]
        self._add_files(paths)

    # ── Browse ────────────────────────────────────────────────────────────────

    def browse(self):
        paths, _ = QFileDialog.getOpenFileNames(
            self,
            "Select video files",
            "",
            "Video files (*.mp4 *.mkv *.avi *.mov *.wmv *.flv *.m4v *.mpg *.mpeg *.ts *.webm *.rmvb *.rm);;All files (*)",
        )
        if paths:
            self._add_files(paths)

    # ── Internals ─────────────────────────────────────────────────────────────

    def _add_files(self, paths: List[str]):
        added = False
        for p in paths:
            ext = os.path.splitext(p)[1].lower()
            if ext in VIDEO_EXTENSIONS and p not in self._paths:
                self._paths.append(p)
                added = True
        if added:
            self._refresh_list()
            self.files_changed.emit(self._paths)

    def _refresh_list(self):
        # Remove all existing file rows (leave stretch)
        while self._list_layout.count() > 1:
            item = self._list_layout.takeAt(0)
            if item.widget():
                item.widget().deleteLater()

        for path in self._paths:
            row = _FileRow(path)
            row.removed.connect(self._remove_file)
            self._list_layout.insertWidget(self._list_layout.count() - 1, row)

        self._scroll.setVisible(bool(self._paths))
        hint = "Drop more files  or  Browse" if self._paths else "Drop video files here  or  Browse"
        self._hint_label.setText(hint)

    def _remove_file(self, path: str):
        if path in self._paths:
            self._paths.remove(path)
            self._refresh_list()
            self.files_changed.emit(self._paths)
