from __future__ import annotations

import json
import uuid
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from app.config import Settings, get_settings
from app.main import app

TEMPLATES_DIR = Path(__file__).resolve().parent.parent / "templates"


@pytest.fixture()
def settings(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Settings:
    output_dir = tmp_path / "output"
    output_dir.mkdir()
    monkeypatch.setenv("DOCUMENT_ENGINE_OUTPUT_DIR", str(output_dir))
    monkeypatch.setenv("DOCUMENT_ENGINE_TEMPLATE_DIR", str(TEMPLATES_DIR))
    monkeypatch.setenv("DOCUMENT_ENGINE_INTERNAL_SECRET", "test-secret")
    monkeypatch.setenv("DOCUMENT_ENGINE_PDF_ENABLED", "false")
    get_settings.cache_clear()
    s = get_settings()
    yield s
    get_settings.cache_clear()


@pytest.fixture()
def client(settings: Settings) -> TestClient:
    return TestClient(app)


@pytest.fixture()
def auth_headers() -> dict[str, str]:
    return {"X-Internal-Secret": "test-secret"}


@pytest.fixture()
def valid_spec() -> dict:
    return {
        "spec_id": str(uuid.uuid4()),
        "run_id": str(uuid.uuid4()),
        "task_id": str(uuid.uuid4()),
        "document_type": "solicitacao_aquisicao_ti",
        "template_key": "solicitacao_aquisicao_ti",
        "template_version": "1.0",
        "title": "Solicitação de Aquisição de TI",
        "metadata": {
            "generated_at": "2026-09-12T10:30:00Z",
            "employee_id": str(uuid.uuid4()),
            "employee_name": "João Silva",
            "employee_re": "123456",
            "department": "TI - Infraestrutura",
        },
        "fields": {
            "numero_solicitacao": "SA-2026-0042",
            "data_solicitacao": "2026-09-12",
            "solicitante": "João Silva",
            "area_solicitante": "TI - Infraestrutura",
            "fornecedor": "Dell Technologies",
            "valor": "R$ 45.000,00",
            "justificativa": "Substituição de equipamentos obsoletos",
            "categoria_produto": "Notebooks",
            "aprovacao_cade": "Não aplicável",
            "observacoes": "",
        },
        "sections": [],
        "rules": [],
        "sources": [],
        "output": {
            "formats": ["docx"],
            "filename_prefix": "SA-2026-0042",
        },
    }
