from __future__ import annotations

from pathlib import Path

import pytest

from app.services.validators import (
    ValidationError,
    sanitize_filename,
    validate_docx,
    validate_pdf,
)

TEMPLATES_DIR = Path(__file__).resolve().parent.parent / "templates"
VALID_DOCX = TEMPLATES_DIR / "solicitacao_aquisicao_ti" / "v1.0" / "template.docx"


class TestValidateDocx:
    def test_valid_docx_passes(self):
        validate_docx(VALID_DOCX)

    def test_nonexistent_file(self, tmp_path: Path):
        with pytest.raises(ValidationError) as exc_info:
            validate_docx(tmp_path / "missing.docx")
        assert exc_info.value.code == "INVALID_OUTPUT"

    def test_wrong_extension(self, tmp_path: Path):
        f = tmp_path / "file.txt"
        f.write_bytes(b"PK" + b"\x00" * 2000)
        with pytest.raises(ValidationError) as exc_info:
            validate_docx(f)
        assert exc_info.value.code == "INVALID_OUTPUT"

    def test_empty_file(self, tmp_path: Path):
        f = tmp_path / "empty.docx"
        f.write_bytes(b"")
        with pytest.raises(ValidationError) as exc_info:
            validate_docx(f)
        assert exc_info.value.code == "EMPTY_OUTPUT"

    def test_too_small_file(self, tmp_path: Path):
        f = tmp_path / "small.docx"
        f.write_bytes(b"x" * 100)
        with pytest.raises(ValidationError) as exc_info:
            validate_docx(f)
        assert exc_info.value.code == "INVALID_OUTPUT"

    def test_not_a_zip(self, tmp_path: Path):
        f = tmp_path / "notzip.docx"
        f.write_bytes(b"NOT A ZIP" * 200)
        with pytest.raises(ValidationError) as exc_info:
            validate_docx(f)
        assert exc_info.value.code == "INVALID_OUTPUT"

    def test_path_outside_allowed_dir(self, tmp_path: Path):
        allowed = tmp_path / "allowed"
        allowed.mkdir()
        with pytest.raises(ValidationError) as exc_info:
            validate_docx(VALID_DOCX, allowed_dir=allowed)
        assert exc_info.value.code == "INVALID_OUTPUT"


class TestValidatePdf:
    def test_nonexistent_file(self, tmp_path: Path):
        with pytest.raises(ValidationError) as exc_info:
            validate_pdf(tmp_path / "missing.pdf")
        assert exc_info.value.code == "INVALID_OUTPUT"

    def test_wrong_extension(self, tmp_path: Path):
        f = tmp_path / "file.txt"
        f.write_bytes(b"%PDF-1.4" + b"\x00" * 200)
        with pytest.raises(ValidationError) as exc_info:
            validate_pdf(f)
        assert exc_info.value.code == "INVALID_OUTPUT"

    def test_empty_file(self, tmp_path: Path):
        f = tmp_path / "empty.pdf"
        f.write_bytes(b"")
        with pytest.raises(ValidationError) as exc_info:
            validate_pdf(f)
        assert exc_info.value.code == "EMPTY_OUTPUT"

    def test_invalid_header(self, tmp_path: Path):
        f = tmp_path / "bad.pdf"
        f.write_bytes(b"NOT PDF" * 50)
        with pytest.raises(ValidationError) as exc_info:
            validate_pdf(f)
        assert exc_info.value.code == "INVALID_OUTPUT"

    def test_valid_pdf_header(self, tmp_path: Path):
        f = tmp_path / "good.pdf"
        f.write_bytes(b"%PDF-1.4" + b"\x00" * 200)
        validate_pdf(f)


class TestSanitizeFilename:
    def test_normal_name(self):
        assert sanitize_filename("SA-2026-0042") == "SA-2026-0042"

    def test_path_traversal_stripped(self):
        result = sanitize_filename("../../etc/passwd")
        assert "/" not in result
        assert ".." not in result

    def test_special_chars_stripped(self):
        result = sanitize_filename("file name (1).docx")
        assert "(" not in result
        assert " " not in result

    def test_empty_becomes_document(self):
        assert sanitize_filename("") == "document"
        assert sanitize_filename("!!!") == "document"

    def test_max_length(self):
        result = sanitize_filename("a" * 500)
        assert len(result) <= 200
