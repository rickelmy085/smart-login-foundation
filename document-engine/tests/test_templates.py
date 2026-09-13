from __future__ import annotations

import json
from pathlib import Path

import pytest

from app.services.templates import Manifest, TemplateError, TemplateLoader

TEMPLATES_DIR = Path(__file__).resolve().parent.parent / "templates"


@pytest.fixture()
def loader() -> TemplateLoader:
    return TemplateLoader(TEMPLATES_DIR)


class TestTemplateLoader:
    def test_resolve_existing_template(self, loader: TemplateLoader):
        resolved = loader.resolve("solicitacao_aquisicao_ti", "1.0")
        assert resolved.key == "solicitacao_aquisicao_ti"
        assert resolved.version == "1.0"
        assert resolved.docx_path.is_file()
        assert resolved.manifest.template_key == "solicitacao_aquisicao_ti"

    def test_template_not_found(self, loader: TemplateLoader):
        with pytest.raises(TemplateError) as exc_info:
            loader.resolve("nonexistent_template", "1.0")
        assert exc_info.value.code == "TEMPLATE_NOT_FOUND"

    def test_version_not_found(self, loader: TemplateLoader):
        with pytest.raises(TemplateError) as exc_info:
            loader.resolve("solicitacao_aquisicao_ti", "99.0")
        assert exc_info.value.code == "TEMPLATE_NOT_FOUND"

    def test_invalid_key_rejected(self, loader: TemplateLoader):
        with pytest.raises(TemplateError) as exc_info:
            loader.resolve("../etc/passwd", "1.0")
        assert exc_info.value.code == "TEMPLATE_NOT_FOUND"

    def test_invalid_version_rejected(self, loader: TemplateLoader):
        with pytest.raises(TemplateError) as exc_info:
            loader.resolve("solicitacao_aquisicao_ti", "../bad")
        assert exc_info.value.code == "TEMPLATE_VERSION_NOT_FOUND"

    def test_path_traversal_blocked(self, loader: TemplateLoader):
        with pytest.raises(TemplateError):
            loader.resolve("../../etc", "1.0")

    def test_absolute_path_key_blocked(self, loader: TemplateLoader):
        with pytest.raises(TemplateError):
            loader.resolve("/etc/passwd", "1.0")

    def test_list_templates(self, loader: TemplateLoader):
        keys = loader.list_templates()
        assert "solicitacao_aquisicao_ti" in keys

    def test_validate_required_fields_all_present(self, loader: TemplateLoader):
        resolved = loader.resolve("solicitacao_aquisicao_ti", "1.0")
        fields = {name: "value" for name in resolved.manifest.required_fields}
        missing = loader.validate_required_fields(resolved.manifest, fields)
        assert missing == []

    def test_validate_required_fields_missing(self, loader: TemplateLoader):
        resolved = loader.resolve("solicitacao_aquisicao_ti", "1.0")
        missing = loader.validate_required_fields(resolved.manifest, {})
        assert len(missing) > 0
        assert "solicitante" in missing

    def test_validate_required_fields_empty_string_is_missing(self, loader: TemplateLoader):
        resolved = loader.resolve("solicitacao_aquisicao_ti", "1.0")
        fields = {name: "" for name in resolved.manifest.required_fields}
        missing = loader.validate_required_fields(resolved.manifest, fields)
        assert len(missing) == len(resolved.manifest.required_fields)


class TestManifestValidation:
    def test_manifest_key_mismatch(self, tmp_path: Path):
        tpl_dir = tmp_path / "bad_key" / "v1.0"
        tpl_dir.mkdir(parents=True)
        (tpl_dir / "template.docx").write_bytes(b"PK fake")
        manifest = {
            "template_key": "different_key",
            "template_version": "1.0",
            "document_type": "bad_key",
            "name": "Bad",
        }
        (tpl_dir / "manifest.json").write_text(json.dumps(manifest))

        loader = TemplateLoader(tmp_path)
        with pytest.raises(TemplateError) as exc_info:
            loader.resolve("bad_key", "1.0")
        assert "does not match" in exc_info.value.message

    def test_manifest_version_mismatch(self, tmp_path: Path):
        tpl_dir = tmp_path / "bad_ver" / "v1.0"
        tpl_dir.mkdir(parents=True)
        (tpl_dir / "template.docx").write_bytes(b"PK fake")
        manifest = {
            "template_key": "bad_ver",
            "template_version": "2.0",
            "document_type": "bad_ver",
            "name": "Bad",
        }
        (tpl_dir / "manifest.json").write_text(json.dumps(manifest))

        loader = TemplateLoader(tmp_path)
        with pytest.raises(TemplateError) as exc_info:
            loader.resolve("bad_ver", "1.0")
        assert exc_info.value.code == "TEMPLATE_VERSION_NOT_FOUND"
