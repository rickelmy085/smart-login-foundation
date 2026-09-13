from __future__ import annotations

import base64
import hashlib
import logging
import time
from pathlib import Path

from fastapi import Depends, FastAPI, HTTPException

from app.config import get_settings
from app.middleware.auth import verify_internal_secret
from app.schemas import (
    DocumentSpec,
    ErrorResponse,
    FileOutput,
    GenerateResponse,
    GenerationWarning,
    HealthResponse,
    OutputFormat,
    TemplateInfo,
)
from app.services.docx import DocxService
from app.services.pdf import PdfError, PdfService
from app.services.templates import TemplateError, TemplateLoader
from app.services.validators import (
    ValidationError,
    sanitize_filename,
    validate_docx,
    validate_pdf,
)

logger = logging.getLogger(__name__)

app = FastAPI(
    title="ABIS Document Engine",
    version="1.0.0",
    description="Professional document generation service for ABIS",
)


def _sha256(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            h.update(chunk)
    return h.hexdigest()


@app.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    settings = get_settings()
    loader = TemplateLoader(settings.resolve_template_dir())
    keys = loader.list_templates()
    return HealthResponse(
        status="ok",
        service="document-engine",
        version="1.0.0",
        templates_loaded=len(keys),
        template_keys=keys,
    )


@app.post(
    "/generate",
    response_model=GenerateResponse,
    responses={
        400: {"model": ErrorResponse},
        401: {"model": ErrorResponse},
        403: {"model": ErrorResponse},
        404: {"model": ErrorResponse},
        500: {"model": ErrorResponse},
    },
)
async def generate(
    spec: DocumentSpec,
    _secret: str = Depends(verify_internal_secret),
) -> GenerateResponse:
    start_time = time.monotonic()
    settings = get_settings()

    logger.info(
        "generate: spec_id=%s run_id=%s template=%s v=%s",
        spec.spec_id,
        spec.run_id,
        spec.template_key,
        spec.template_version,
    )

    loader = TemplateLoader(settings.resolve_template_dir())

    try:
        resolved = loader.resolve(spec.template_key, spec.template_version)
    except TemplateError as exc:
        raise HTTPException(
            status_code=404,
            detail={
                "status": "error",
                "error_code": exc.code,
                "message": exc.message,
                "spec_id": spec.spec_id,
                "run_id": spec.run_id,
            },
        )

    missing = loader.validate_required_fields(resolved.manifest, spec.fields)
    if missing:
        raise HTTPException(
            status_code=400,
            detail={
                "status": "error",
                "error_code": "MISSING_REQUIRED_FIELD",
                "message": f"Required fields missing: {', '.join(missing)}",
                "spec_id": spec.spec_id,
                "run_id": spec.run_id,
                "details": {"missing_fields": missing},
            },
        )

    output_base = settings.resolve_output_dir() / spec.run_id
    output_base.mkdir(parents=True, exist_ok=True)

    prefix = sanitize_filename(spec.output.filename_prefix)

    docx_filename = f"{prefix}.docx"
    docx_path = output_base / docx_filename

    docx_service = DocxService()

    try:
        docx_service.generate(
            template=resolved,
            fields=spec.fields,
            output_path=docx_path,
            title=spec.title,
        )
    except Exception as exc:
        logger.error(
            "DOCX generation failed: spec_id=%s error=%s",
            spec.spec_id,
            str(exc),
        )
        raise HTTPException(
            status_code=500,
            detail={
                "status": "error",
                "error_code": "GENERATION_FAILED",
                "message": f"DOCX generation failed: {str(exc)}",
                "spec_id": spec.spec_id,
                "run_id": spec.run_id,
            },
        )

    try:
        validate_docx(docx_path, allowed_dir=settings.resolve_output_dir())
    except ValidationError as exc:
        raise HTTPException(
            status_code=500,
            detail={
                "status": "error",
                "error_code": exc.code,
                "message": exc.message,
                "spec_id": spec.spec_id,
                "run_id": spec.run_id,
            },
        )

    files: list[FileOutput] = []
    warnings: list[GenerationWarning] = []

    if OutputFormat.DOCX in spec.output.formats:
        docx_bytes = docx_path.read_bytes()
        files.append(
            FileOutput(
                format="docx",
                filename=docx_filename,
                size_bytes=len(docx_bytes),
                sha256=_sha256(docx_path),
                data_base64=base64.b64encode(docx_bytes).decode("ascii"),
            )
        )

    if OutputFormat.PDF in spec.output.formats:
        pdf_filename = f"{prefix}.pdf"
        pdf_path = output_base / pdf_filename
        pdf_service = PdfService()

        if not pdf_service.is_available:
            warnings.append(
                GenerationWarning(
                    code="PDF_UNAVAILABLE",
                    message="PDF generation is not available (LibreOffice not found or disabled)",
                )
            )
        else:
            try:
                pdf_service.generate(docx_path, pdf_path)
                validate_pdf(pdf_path, allowed_dir=settings.resolve_output_dir())
                pdf_bytes = pdf_path.read_bytes()
                files.append(
                    FileOutput(
                        format="pdf",
                        filename=pdf_filename,
                        size_bytes=len(pdf_bytes),
                        sha256=_sha256(pdf_path),
                        data_base64=base64.b64encode(pdf_bytes).decode("ascii"),
                    )
                )
            except (PdfError, ValidationError) as exc:
                warnings.append(
                    GenerationWarning(
                        code=exc.code,
                        message=exc.message,
                    )
                )

    elapsed = time.monotonic() - start_time
    logger.info(
        "generate: spec_id=%s completed in %.2fs files=%d warnings=%d",
        spec.spec_id,
        elapsed,
        len(files),
        len(warnings),
    )

    return GenerateResponse(
        status="generated",
        spec_id=spec.spec_id,
        run_id=spec.run_id,
        template_used=TemplateInfo(
            key=resolved.key,
            version=resolved.version,
            path=str(resolved.docx_path),
        ),
        files=files,
        warnings=warnings,
    )


@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    logger.error("Unhandled exception: %s", str(exc))
    return {
        "status": "error",
        "error_code": "INTERNAL_ERROR",
        "message": "An internal error occurred",
    }
