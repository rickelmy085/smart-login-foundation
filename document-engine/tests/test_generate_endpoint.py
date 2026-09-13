from __future__ import annotations

import base64
import copy
import uuid

import pytest
from fastapi.testclient import TestClient


class TestGenerateEndpoint:
    def test_generate_without_auth_returns_401(self, client: TestClient, valid_spec: dict):
        response = client.post("/generate", json=valid_spec)
        assert response.status_code == 401

    def test_generate_with_wrong_secret_returns_403(
        self, client: TestClient, valid_spec: dict
    ):
        response = client.post(
            "/generate",
            json=valid_spec,
            headers={"X-Internal-Secret": "wrong-secret"},
        )
        assert response.status_code == 403

    def test_generate_with_valid_spec(
        self, client: TestClient, valid_spec: dict, auth_headers: dict
    ):
        response = client.post(
            "/generate", json=valid_spec, headers=auth_headers
        )
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "generated"
        assert data["spec_id"] == valid_spec["spec_id"]
        assert data["run_id"] == valid_spec["run_id"]
        assert data["template_used"]["key"] == "solicitacao_aquisicao_ti"
        assert data["template_used"]["version"] == "1.0"

    def test_generate_returns_docx_file(
        self, client: TestClient, valid_spec: dict, auth_headers: dict
    ):
        response = client.post(
            "/generate", json=valid_spec, headers=auth_headers
        )
        data = response.json()
        docx_files = [f for f in data["files"] if f["format"] == "docx"]
        assert len(docx_files) == 1
        f = docx_files[0]
        assert f["filename"].endswith(".docx")
        assert f["size_bytes"] > 1000
        assert len(f["sha256"]) == 64
        assert f["data_base64"] is not None

        raw = base64.b64decode(f["data_base64"])
        assert raw[:2] == b"PK"

    def test_generate_invalid_spec_returns_422(
        self, client: TestClient, auth_headers: dict
    ):
        response = client.post(
            "/generate",
            json={"bad": "data"},
            headers=auth_headers,
        )
        assert response.status_code == 422

    def test_generate_nonexistent_template_returns_404(
        self, client: TestClient, valid_spec: dict, auth_headers: dict
    ):
        spec = copy.deepcopy(valid_spec)
        spec["template_key"] = "nonexistent"
        spec["document_type"] = "nonexistent"
        response = client.post(
            "/generate", json=spec, headers=auth_headers
        )
        assert response.status_code == 404

    def test_generate_missing_required_field_returns_400(
        self, client: TestClient, valid_spec: dict, auth_headers: dict
    ):
        spec = copy.deepcopy(valid_spec)
        spec["fields"]["solicitante"] = ""
        response = client.post(
            "/generate", json=spec, headers=auth_headers
        )
        assert response.status_code == 400
        data = response.json()
        assert "MISSING_REQUIRED_FIELD" in str(data)

    def test_generate_pdf_warning_when_unavailable(
        self, client: TestClient, valid_spec: dict, auth_headers: dict
    ):
        spec = copy.deepcopy(valid_spec)
        spec["output"]["formats"] = ["docx", "pdf"]
        response = client.post(
            "/generate", json=spec, headers=auth_headers
        )
        assert response.status_code == 200
        data = response.json()
        if not any(f["format"] == "pdf" for f in data["files"]):
            warning_codes = [w["code"] for w in data["warnings"]]
            assert "PDF_UNAVAILABLE" in warning_codes or "PDF_GENERATION_FAILED" in warning_codes

    def test_generate_invalid_uuid_returns_422(
        self, client: TestClient, valid_spec: dict, auth_headers: dict
    ):
        spec = copy.deepcopy(valid_spec)
        spec["spec_id"] = "not-a-uuid"
        response = client.post(
            "/generate", json=spec, headers=auth_headers
        )
        assert response.status_code == 422


class TestE2EStandalone:
    def test_full_generation_flow(
        self, client: TestClient, valid_spec: dict, auth_headers: dict
    ):
        health = client.get("/health")
        assert health.status_code == 200
        assert "solicitacao_aquisicao_ti" in health.json()["template_keys"]

        response = client.post(
            "/generate", json=valid_spec, headers=auth_headers
        )
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "generated"
        assert len(data["files"]) >= 1

        docx_file = next(f for f in data["files"] if f["format"] == "docx")
        raw = base64.b64decode(docx_file["data_base64"])
        assert raw[:2] == b"PK"
        assert docx_file["size_bytes"] == len(raw)

        from docx import Document
        import io
        doc = Document(io.BytesIO(raw))
        full_text = "\n".join(p.text for p in doc.paragraphs)
        assert "João Silva" in full_text
        assert "Dell Technologies" in full_text
        assert "{{solicitante}}" not in full_text
        assert "{{fornecedor}}" not in full_text
