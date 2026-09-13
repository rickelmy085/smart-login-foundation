from __future__ import annotations

import hmac
import logging

from fastapi import HTTPException, Security, status
from fastapi.security import APIKeyHeader

from app.config import get_settings

logger = logging.getLogger(__name__)

_internal_secret_header = APIKeyHeader(name="X-Internal-Secret", auto_error=False)


async def verify_internal_secret(
    secret: str | None = Security(_internal_secret_header),
) -> str:
    settings = get_settings()

    if not settings.internal_secret:
        logger.error("DOCUMENT_ENGINE_INTERNAL_SECRET is not configured")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal authentication is not configured",
        )

    if not secret:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing internal authentication header",
        )

    if not hmac.compare_digest(secret, settings.internal_secret):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Invalid internal authentication credentials",
        )

    return secret
