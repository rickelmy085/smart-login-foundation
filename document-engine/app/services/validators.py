from __future__ import annotations

import logging
import re
import zipfile
from pathlib import Path

logger = logging.getLogger(__name__)

_PLACEHOLDER_RE = re.compile(r"\{\{([a-zA-Z0-9_]+)\}\}")

_REQUIRED_DOCX_PARTS = [
    "[Content_Types].xml",
    "word/document.xml",
]


class ValidationError(Exception):
    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(message)


def validate_docx(path: Path, allowed_dir: Path | None = None) -> None:
    if not path.is_file():
        raise ValidationError("INVALID_OUTPUT", f"DOCX file does not exist: {path}")

    if allowed_dir is not None:
        try:
            path.resolve().relative_to(allowed_dir.resolve())
        except ValueError:
            raise ValidationError(
                "INVALID_OUTPUT",
                "DOCX file is outside the allowed output directory",
            )

    if path.suffix.lower() != ".docx":
        raise ValidationError("INVALID_OUTPUT", "File does not have .docx extension")

    size = path.stat().st_size
    if size == 0:
        raise ValidationError("EMPTY_OUTPUT", "DOCX file is empty")

    if size < 1000:
        raise ValidationError(
            "INVALID_OUTPUT",
            f"DOCX file is suspiciously small ({size} bytes)",
        )

    if not zipfile.is_zipfile(str(path)):
        raise ValidationError("INVALID_OUTPUT", "DOCX is not a valid ZIP archive")

    with zipfile.ZipFile(str(path), "r") as zf:
        names = zf.namelist()
        for required in _REQUIRED_DOCX_PARTS:
            if required not in names:
                raise ValidationError(
                    "INVALID_OUTPUT",
                    f"DOCX is missing required part: {required}",
                )

        try:
            document_xml = zf.read("word/document.xml").decode("utf-8", errors="replace")
        except KeyError:
            raise ValidationError(
                "INVALID_OUTPUT",
                "Cannot read word/document.xml from DOCX",
            )

    remaining = _PLACEHOLDER_RE.findall(document_xml)
    if remaining:
        logger.warning(
            "Unsubstituted placeholders found in DOCX: %s",
            list(set(remaining)),
        )


def validate_pdf(path: Path, allowed_dir: Path | None = None) -> None:
    if not path.is_file():
        raise ValidationError("INVALID_OUTPUT", f"PDF file does not exist: {path}")

    if allowed_dir is not None:
        try:
            path.resolve().relative_to(allowed_dir.resolve())
        except ValueError:
            raise ValidationError(
                "INVALID_OUTPUT",
                "PDF file is outside the allowed output directory",
            )

    if path.suffix.lower() != ".pdf":
        raise ValidationError("INVALID_OUTPUT", "File does not have .pdf extension")

    size = path.stat().st_size
    if size == 0:
        raise ValidationError("EMPTY_OUTPUT", "PDF file is empty")

    if size < 100:
        raise ValidationError(
            "INVALID_OUTPUT",
            f"PDF file is suspiciously small ({size} bytes)",
        )

    header = path.read_bytes()[:5]
    if not header.startswith(b"%PDF-"):
        raise ValidationError(
            "INVALID_OUTPUT",
            "File does not have a valid PDF header",
        )


def sanitize_filename(name: str) -> str:
    clean = "".join(c for c in name if c.isalnum() or c in "-_.")
    clean = clean.replace("..", "")
    if not clean:
        clean = "document"
    return clean[:200]
