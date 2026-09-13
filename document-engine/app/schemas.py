from __future__ import annotations

import re
from datetime import datetime
from enum import Enum
from typing import Any

from pydantic import BaseModel, ConfigDict, Field, field_validator

_UUID_RE = re.compile(
    r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
)
_SAFE_KEY_RE = re.compile(r"^[a-zA-Z0-9_-]+$")
_VERSION_RE = re.compile(r"^[0-9]+(\.[0-9]+)*$")


class OutputFormat(str, Enum):
    DOCX = "docx"
    PDF = "pdf"


class MetadataBlock(BaseModel):
    model_config = ConfigDict(extra="allow")

    generated_at: datetime | None = None
    employee_id: str | None = None
    employee_name: str | None = None
    employee_re: str | None = None
    department: str | None = None


class Section(BaseModel):
    model_config = ConfigDict(extra="allow")

    id: str
    type: str
    content: dict[str, Any] = Field(default_factory=dict)


class RuleSource(BaseModel):
    model_config = ConfigDict(extra="allow")

    document: str | None = None
    chunk_id: int | str | None = None
    excerpt: str | None = None


class Rule(BaseModel):
    model_config = ConfigDict(extra="allow")

    rule_id: str | None = None
    rule: str
    required: bool = False
    reason: str | None = None
    source: RuleSource | None = None


class Source(BaseModel):
    model_config = ConfigDict(extra="allow")

    document: str | None = None
    chunk_id: int | str | None = None
    snippet: str | None = None
    requirement: str | None = None


class OutputConfig(BaseModel):
    model_config = ConfigDict(extra="ignore")

    formats: list[OutputFormat] = Field(default_factory=lambda: [OutputFormat.DOCX])
    filename_prefix: str = "document"

    @field_validator("formats")
    @classmethod
    def at_least_one_format(cls, v: list[OutputFormat]) -> list[OutputFormat]:
        if not v:
            raise ValueError("At least one output format must be specified")
        return v

    @field_validator("filename_prefix")
    @classmethod
    def safe_filename_prefix(cls, v: str) -> str:
        if not v or not v.strip():
            return "document"
        cleaned = "".join(c for c in v if c.isalnum() or c in "-_.")
        cleaned = cleaned.replace("..", "")
        return cleaned[:200] or "document"


class DocumentSpec(BaseModel):
    model_config = ConfigDict(extra="ignore")

    spec_id: str
    run_id: str
    task_id: str
    document_type: str
    template_key: str
    template_version: str
    title: str = ""
    metadata: MetadataBlock = Field(default_factory=MetadataBlock)
    fields: dict[str, str] = Field(default_factory=dict)
    sections: list[Section] = Field(default_factory=list)
    rules: list[Rule] = Field(default_factory=list)
    sources: list[Source] = Field(default_factory=list)
    output: OutputConfig = Field(default_factory=OutputConfig)

    @field_validator("spec_id", "run_id", "task_id")
    @classmethod
    def validate_uuid(cls, v: str) -> str:
        if not _UUID_RE.match(v):
            raise ValueError(f"Invalid UUID format: {v!r}")
        return v

    @field_validator("template_key")
    @classmethod
    def validate_template_key(cls, v: str) -> str:
        if not v or not _SAFE_KEY_RE.match(v):
            raise ValueError(f"Invalid template key: {v!r}")
        return v

    @field_validator("template_version")
    @classmethod
    def validate_template_version(cls, v: str) -> str:
        if not v or not _VERSION_RE.match(v):
            raise ValueError(f"Invalid template version: {v!r}")
        return v

    @field_validator("document_type")
    @classmethod
    def validate_document_type(cls, v: str) -> str:
        if not v or not _SAFE_KEY_RE.match(v):
            raise ValueError(f"Invalid document type: {v!r}")
        return v


class TemplateInfo(BaseModel):
    key: str
    version: str
    path: str | None = None


class FileOutput(BaseModel):
    format: str
    filename: str
    size_bytes: int
    sha256: str
    data_base64: str | None = None


class GenerationWarning(BaseModel):
    code: str
    message: str


class GenerateResponse(BaseModel):
    status: str = "generated"
    spec_id: str
    run_id: str
    template_used: TemplateInfo
    files: list[FileOutput] = Field(default_factory=list)
    warnings: list[GenerationWarning] = Field(default_factory=list)


class HealthResponse(BaseModel):
    status: str = "ok"
    service: str = "document-engine"
    version: str
    templates_loaded: int = 0
    template_keys: list[str] = Field(default_factory=list)


class ErrorResponse(BaseModel):
    status: str = "error"
    error_code: str
    message: str
    spec_id: str | None = None
    run_id: str | None = None
    details: dict[str, Any] | None = None
