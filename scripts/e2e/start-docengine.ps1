#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Starts the Document Engine Python service.
.DESCRIPTION
    Runs the FastAPI Document Engine service on the configured port.
#>

param(
    [string]$Port = "8090"
)

Set-Location (Split-Path -Parent $PSScriptRoot) -Parent
Set-Location "document-engine"

Write-Host "Starting Document Engine on port $Port..." -ForegroundColor Cyan

.\.venv\Scripts\Activate.ps1
python -m uvicorn app.main:app --host 127.0.0.1 --port $Port --log-level info