from __future__ import annotations

import uuid

import pytest
from pydantic import ValidationError

from app.schemas import DocumentSpec, OutputConfig, OutputFormat


def _minimal_spec(**overrides) -> dict:
    base = {
        "spec_id": str(uuid.uuid4()),
        "run_id": str(uuid.uuid4()),
        "task_id": str(uuid.uuid4()),
        "document_type": "solicitacao_aquisicao_ti",
        "template_key": "solicitacao_aquisicao_ti",
        "template_version": "1.0",
    }
    base.update(overrides)
    return base


class TestDocumentSpec:
    def test_valid_spec(self):
        spec = DocumentSpec(**_minimal_spec())
        assert spec.template_key == "solicitacao_aquisicao_ti"
        assert spec.template_version == "1.0"

    def test_invalid_uuid(self):
        with pytest.raises(ValidationError, match="Invalid UUID"):
            DocumentSpec(**_minimal_spec(spec_id="not-a-uuid"))

    def test_invalid_template_key(self):
        with pytest.raises(ValidationError, match="Invalid template key"):
            DocumentSpec(**_minimal_spec(template_key="../bad"))

    def test_invalid_template_version(self):
        with pytest.raises(ValidationError, match="Invalid template version"):
            DocumentSpec(**_minimal_spec(template_version="abc"))

    def test_invalid_document_type(self):
        with pytest.raises(ValidationError, match="Invalid document type"):
            DocumentSpec(**_minimal_spec(document_type="has spaces"))

    def test_missing_required_fields(self):
        with pytest.raises(ValidationError):
            DocumentSpec()

    def test_fields_default_empty(self):
        spec = DocumentSpec(**_minimal_spec())
        assert spec.fields == {}

    def test_sections_rules_sources_default_empty(self):
        spec = DocumentSpec(**_minimal_spec())
        assert spec.sections == []
        assert spec.rules == []
        assert spec.sources == []


class TestOutputConfig:
    def test_default_format_is_docx(self):
        cfg = OutputConfig()
        assert cfg.formats == [OutputFormat.DOCX]

    def test_empty_formats_rejected(self):
        with pytest.raises(ValidationError, match="At least one"):
            OutputConfig(formats=[])

    def test_filename_prefix_sanitized(self):
        cfg = OutputConfig(filename_prefix="../evil/path")
        assert "/" not in cfg.filename_prefix
        assert ".." not in cfg.filename_prefix

    def test_empty_filename_prefix_defaults(self):
        cfg = OutputConfig(filename_prefix="")
        assert cfg.filename_prefix == "document"
