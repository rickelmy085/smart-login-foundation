from __future__ import annotations

import pytest
from fastapi.testclient import TestClient


class TestHealthEndpoint:
    def test_health_returns_200(self, client: TestClient):
        response = client.get("/health")
        assert response.status_code == 200

    def test_health_response_shape(self, client: TestClient):
        response = client.get("/health")
        data = response.json()
        assert data["status"] == "ok"
        assert data["service"] == "document-engine"
        assert "version" in data
        assert "templates_loaded" in data
        assert isinstance(data["template_keys"], list)

    def test_health_lists_templates(self, client: TestClient):
        response = client.get("/health")
        data = response.json()
        assert "solicitacao_aquisicao_ti" in data["template_keys"]

    def test_health_no_auth_required(self, client: TestClient):
        response = client.get("/health")
        assert response.status_code == 200
