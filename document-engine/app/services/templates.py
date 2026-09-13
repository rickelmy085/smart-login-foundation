from __future__ import annotations

import json
import logging
import re
from dataclasses import dataclass, field
from pathlib import Path

from app.config import get_settings

logger = logging.getLogger(__name__)

_SAFE_KEY_RE = re.compile(r"^[a-zA-Z0-9_-]+$")
_SAFE_VERSION_RE = re.compile(r"^[0-9]+(\.[0-9]+)*$")


@dataclass
class Manifest:
    template_key: str
    template_version: str
    document_type: str
    name: str
    required_fields: list[str] = field(default_factory=list)
    optional_fields: list[str] = field(default_factory=list)
    formats: list[str] = field(default_factory=lambda: ["docx"])

    @classmethod
    def from_dict(cls, data: dict) -> Manifest:
        return cls(
            template_key=str(data.get("template_key", "")),
            template_version=str(data.get("template_version", "")),
            document_type=str(data.get("document_type", "")),
            name=str(data.get("name", "")),
            required_fields=list(data.get("required_fields", [])),
            optional_fields=list(data.get("optional_fields", [])),
            formats=list(data.get("formats", ["docx"])),
        )


@dataclass
class ResolvedTemplate:
    key: str
    version: str
    docx_path: Path
    manifest: Manifest


class TemplateError(Exception):
    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(message)


class TemplateLoader:
    def __init__(self, base_dir: Path | None = None) -> None:
        if base_dir is None:
            base_dir = get_settings().resolve_template_dir()
        self._base_dir = base_dir.resolve()

    def _validate_key(self, template_key: str) -> None:
        if not template_key or not _SAFE_KEY_RE.match(template_key):
            raise TemplateError(
                "TEMPLATE_NOT_FOUND",
                f"Invalid template key: {template_key!r}",
            )

    def _validate_version(self, version: str) -> None:
        if not version or not _SAFE_VERSION_RE.match(version):
            raise TemplateError(
                "TEMPLATE_VERSION_NOT_FOUND",
                f"Invalid template version: {version!r}",
            )

    def _resolve_dir(self, template_key: str, version: str) -> Path:
        self._validate_key(template_key)
        self._validate_version(version)

        candidate = (self._base_dir / template_key / f"v{version}").resolve()

        try:
            candidate.relative_to(self._base_dir)
        except ValueError:
            raise TemplateError(
                "TEMPLATE_NOT_FOUND",
                "Template path escapes the allowed directory",
            )

        return candidate

    def resolve(self, template_key: str, version: str) -> ResolvedTemplate:
        template_dir = self._resolve_dir(template_key, version)

        docx_path = template_dir / "template.docx"
        manifest_path = template_dir / "manifest.json"

        if not template_dir.is_dir():
            raise TemplateError(
                "TEMPLATE_NOT_FOUND",
                f"Template '{template_key}' version '{version}' not found",
            )

        if not docx_path.is_file():
            raise TemplateError(
                "TEMPLATE_NOT_FOUND",
                f"Template file missing for '{template_key}' v{version}",
            )

        manifest = self._load_manifest(manifest_path, template_key, version)

        return ResolvedTemplate(
            key=template_key,
            version=version,
            docx_path=docx_path,
            manifest=manifest,
        )

    def _load_manifest(
        self, manifest_path: Path, template_key: str, version: str
    ) -> Manifest:
        if not manifest_path.is_file():
            logger.warning(
                "manifest.json not found for %s v%s, using defaults",
                template_key,
                version,
            )
            return Manifest(
                template_key=template_key,
                template_version=version,
                document_type=template_key,
                name=template_key,
            )

        try:
            raw = manifest_path.read_text(encoding="utf-8")
            data = json.loads(raw)
        except (OSError, json.JSONDecodeError) as exc:
            raise TemplateError(
                "TEMPLATE_NOT_FOUND",
                f"Invalid manifest.json for '{template_key}' v{version}: {exc}",
            )

        manifest = Manifest.from_dict(data)

        if manifest.template_key != template_key:
            raise TemplateError(
                "TEMPLATE_NOT_FOUND",
                "Manifest template_key does not match requested key",
            )

        if manifest.template_version != version:
            raise TemplateError(
                "TEMPLATE_VERSION_NOT_FOUND",
                "Manifest template_version does not match requested version",
            )

        return manifest

    def list_templates(self) -> list[str]:
        keys: list[str] = []
        if not self._base_dir.is_dir():
            return keys
        for entry in sorted(self._base_dir.iterdir()):
            if entry.is_dir() and _SAFE_KEY_RE.match(entry.name):
                keys.append(entry.name)
        return keys

    def validate_required_fields(
        self, manifest: Manifest, fields: dict[str, str]
    ) -> list[str]:
        missing: list[str] = []
        for name in manifest.required_fields:
            value = fields.get(name)
            if value is None or str(value).strip() == "":
                missing.append(name)
        return missing
