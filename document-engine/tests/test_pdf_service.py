from __future__ import annotations

from pathlib import Path

import pytest

from app.services.pdf import PdfError, PdfService


class TestPdfService:
    def test_availability_check(self):
        service = PdfService()
        # On CI or test environments, LibreOffice may not be installed
        # This test just ensures the property works without error
        _ = service.is_available

    def test_generate_fails_when_unavailable(self, tmp_path: Path, monkeypatch):
        monkeypatch.setenv("DOCUMENT_ENGINE_PDF_ENABLED", "false")
        from app.config import get_settings
        get_settings.cache_clear()

        service = PdfService()
        docx_path = tmp_path / "input.docx"
        docx_path.write_bytes(b"PK fake docx")
        pdf_path = tmp_path / "output.pdf"

        with pytest.raises(PdfError) as exc_info:
            service.generate(docx_path, pdf_path)
        assert exc_info.value.code == "PDF_GENERATION_FAILED"

        get_settings.cache_clear()

    def test_generate_fails_when_docx_missing(self, tmp_path: Path):
        service = PdfService()
        if not service.is_available:
            pytest.skip("LibreOffice not available")

        with pytest.raises(PdfError) as exc_info:
            service.generate(tmp_path / "nonexistent.docx", tmp_path / "out.pdf")
        assert exc_info.value.code == "PDF_GENERATION_FAILED"
