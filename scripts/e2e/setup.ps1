#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Sets up the E2E test environment for ABIS.
.DESCRIPTION
    Starts the Document Engine Python service and the Go backend server.
    Runs migrations and seeds the database.
#>

param(
    [string]$BackendPort = "8081",
    [string]$DocEnginePort = "8090",
    [string]$DocEngineUrl = "http://127.0.0.1:8090",
    [string]$DocEngineSecret = "change-me",
    [string]$GroqApiKey = "",
    [switch]$EnableDocumentEngine
)

# Ensure we're in the project root
$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

Write-Host "=== ABIS E2E Environment Setup ===" -ForegroundColor Cyan

# Check if .env exists
$backendEnv = "backend\.env"
if (-not (Test-Path $backendEnv)) {
    Write-Host "Creating backend\.env from example..." -ForegroundColor Yellow
    Copy-Item "backend\.env.example" $backendEnv
}

# Update .env with E2E settings
$envContent = Get-Content $backendEnv -Raw
$envContent = $envContent -replace 'PORT=.*', "PORT=$BackendPort"
$envContent = $envContent -replace 'DOCUMENT_ENGINE_ENABLED=.*', "DOCUMENT_ENGINE_ENABLED=" + $EnableDocumentEngine.ToString().ToLower()
$envContent = $envContent -replace 'DOCUMENT_ENGINE_URL=.*', "DOCUMENT_ENGINE_URL=$DocEngineUrl"
$envContent = $envContent -replace 'DOCUMENT_ENGINE_INTERNAL_SECRET=.*', "DOCUMENT_ENGINE_INTERNAL_SECRET=$DocEngineSecret"
$envContent = $envContent -replace 'DOCUMENT_ENGINE_TIMEOUT_SECONDS=.*', 'DOCUMENT_ENGINE_TIMEOUT_SECONDS=30'
$envContent = $envContent -replace 'DOCUMENT_ENGINE_FALLBACK_TO_GO=.*', 'DOCUMENT_ENGINE_FALLBACK_TO_GO=true'

if ($GroqApiKey) {
    $envContent = $envContent -replace 'GROQ_API_KEY=.*', "GROQ_API_KEY=$GroqApiKey"
}

Set-Content $backendEnv $envContent
Write-Host "Updated backend\.env" -ForegroundColor Green

# Check Document Engine .env
$docEngineEnv = "document-engine\.env"
if (-not (Test-Path $docEngineEnv)) {
    Write-Host "Creating document-engine\.env from example..." -ForegroundColor Yellow
    Copy-Item "document-engine\.env.example" $docEngineEnv
}

$docEnvContent = Get-Content $docEngineEnv -Raw
$docEnvContent = $docEnvContent -replace 'DOCUMENT_ENGINE_HOST=.*', "DOCUMENT_ENGINE_HOST=127.0.0.1"
$docEnvContent = $docEnvContent -replace 'DOCUMENT_ENGINE_PORT=.*', "DOCUMENT_ENGINE_PORT=$DocEnginePort"
$docEnvContent = $docEnvContent -replace 'DOCUMENT_ENGINE_INTERNAL_SECRET=.*', "DOCUMENT_ENGINE_INTERNAL_SECRET=$DocEngineSecret"
$docEnvContent = $docEnvContent -replace 'DOCUMENT_ENGINE_PDF_ENABLED=.*', 'DOCUMENT_ENGINE_PDF_ENABLED=false'
Set-Content $docEngineEnv $docEnvContent
Write-Host "Updated document-engine\.env" -ForegroundColor Green

# Check Python dependencies
Write-Host "Checking Document Engine Python dependencies..." -ForegroundColor Cyan
Set-Location "document-engine"
if (-not (Test-Path ".venv")) {
    Write-Host "Creating Python virtual environment..." -ForegroundColor Yellow
    python -m venv .venv
}
.\.venv\Scripts\Activate.ps1
pip install -q -r requirements.txt
Write-Host "Document Engine dependencies ready" -ForegroundColor Green
Set-Location ..

# Check Go dependencies
Write-Host "Checking Go dependencies..." -ForegroundColor Cyan
Set-Location "backend"
go mod download
go build -o abis.exe .
Write-Host "Go backend built successfully" -ForegroundColor Green
Set-Location ..

# Run migrations
Write-Host "Running database migrations..." -ForegroundColor Cyan
Set-Location "backend"
.\abis.exe --migrate-only 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Migration failed, trying with go run..." -ForegroundColor Yellow
    go run main.go --migrate-only 2>&1 | Out-Null
}
Set-Location ..
Write-Host "Migrations completed" -ForegroundColor Green

Write-Host "=== E2E Environment Ready ===" -ForegroundColor Cyan
Write-Host "Backend: http://127.0.0.1:$BackendPort" -ForegroundColor Green
Write-Host "Document Engine: http://127.0.0.1:$DocEnginePort" -ForegroundColor Green
Write-Host ""
Write-Host "To start services manually:" -ForegroundColor Cyan
Write-Host "  Terminal 1: cd document-engine; .\.venv\Scripts\Activate.ps1; python -m uvicorn app.main:app --host 127.0.0.1 --port $DocEnginePort" -ForegroundColor Gray
Write-Host "  Terminal 2: cd backend; .\abis.exe" -ForegroundColor Gray