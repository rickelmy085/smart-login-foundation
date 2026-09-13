from __future__ import annotations

from pathlib import Path
from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="DOCUMENT_ENGINE_",
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    host: str = "127.0.0.1"
    port: int = 8090
    internal_secret: str = "change-me-in-production"
    output_dir: Path = Path("./output")
    template_dir: Path = Path("./templates")
    pdf_enabled: bool = True
    pdf_timeout_seconds: int = 30
    log_level: str = "INFO"

    def resolve_output_dir(self) -> Path:
        self.output_dir.mkdir(parents=True, exist_ok=True)
        return self.output_dir.resolve()

    def resolve_template_dir(self) -> Path:
        return self.template_dir.resolve()


@lru_cache
def get_settings() -> Settings:
    return Settings()
