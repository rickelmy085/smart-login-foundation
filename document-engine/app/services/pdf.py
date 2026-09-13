from __future__ import annotations

import logging
import shutil
import subprocess
from pathlib import Path

from app.config import get_settings

logger = logging.getLogger(__name__)


def _find_libreoffice() -> str | None:
    candidates = [
        "soffice",
        "libreoffice",
    ]
    for name in candidates:
        path = shutil.which(name)
        if path:
            return path

    win_candidates = [
        Path(r"C:\Program Files\LibreOffice\program\soffice.exe"),
        Path(r"C:\Program Files (x86)\LibreOffice\program\soffice.exe"),
    ]
    for p in win_candidates:
        if p.is_file():
            return str(p)

    return None


class PdfError(Exception):
    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(message)


class PdfService:
    def __init__(self) -> None:
        self._settings = get_settings()
        self._libreoffice_path = _find_libreoffice()

    @property
    def is_available(self) -> bool:
        return self._libreoffice_path is not None and self._settings.pdf_enabled

    def generate(self, docx_path: Path, output_path: Path) -> Path:
        if not self.is_available:
            raise PdfError(
                "PDF_GENERATION_FAILED",
                "PDF generation is not available (LibreOffice not found or disabled)",
            )

        if not docx_path.is_file():
            raise PdfError(
                "PDF_GENERATION_FAILED",
                f"Source DOCX not found: {docx_path}",
            )

        output_path.parent.mkdir(parents=True, exist_ok=True)

        cmd = [
            self._libreoffice_path,
            "--headless",
            "--convert-to",
            "pdf",
            "--outdir",
            str(output_path.parent),
            str(docx_path),
        ]

        timeout = self._settings.pdf_timeout_seconds

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=timeout,
            )
        except subprocess.TimeoutExpired:
            raise PdfError(
                "PDF_GENERATION_FAILED",
                f"LibreOffice conversion timed out after {timeout}s",
            )
        except OSError as exc:
            raise PdfError(
                "PDF_GENERATION_FAILED",
                f"Failed to execute LibreOffice: {exc}",
            )

        expected_pdf = output_path.parent / (docx_path.stem + ".pdf")

        if not expected_pdf.is_file():
            stderr_snippet = (result.stderr or "")[:500]
            raise PdfError(
                "PDF_GENERATION_FAILED",
                f"LibreOffice did not produce a PDF. stderr: {stderr_snippet}",
            )

        if expected_pdf != output_path:
            shutil.move(str(expected_pdf), str(output_path))

        if output_path.stat().st_size == 0:
            output_path.unlink(missing_ok=True)
            raise PdfError(
                "EMPTY_OUTPUT",
                "Generated PDF is empty",
            )

        logger.info(
            "PDF generated: %s size=%d",
            output_path.name,
            output_path.stat().st_size,
        )

        return output_path
