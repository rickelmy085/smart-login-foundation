#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Runs E2E test for Document Engine indisponível (fallback to Go).
#>

param(
    [string]$BackendPort = "8082"
)

$baseUrl = "http://127.0.0.1:$BackendPort"

Write-Host "=== ABIS E2E #6 - Document Engine Indisponível (Fallback) Test ===" -ForegroundColor Cyan
Write-Host ""

# Login
Write-Host "[1/6] Login..." -ForegroundColor Yellow
try {
    $loginBody = @{ re = "123456"; password = "demo123"; remember = $false } | ConvertTo-Json
    $login = Invoke-RestMethod -Uri "$baseUrl/api/login" -Method Post -Body $loginBody -ContentType "application/json" -TimeoutSec 15
    $token = $login.token
    Write-Host "  Login SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Login FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

$headers = @{ Authorization = "Bearer $token" }

# Create Task
Write-Host "[2/6] Create Task..." -ForegroundColor Yellow
try {
    $taskBody = @{ question = "Preciso solicitar aquisição de notebooks Dell para a agência, valor R$ 50.000, justificativa renovação de equipamentos" } | ConvertTo-Json
    $task = Invoke-RestMethod -Uri "$baseUrl/api/tasks" -Method Post -Body $taskBody -ContentType "application/json" -Headers $headers -TimeoutSec 30
    $taskId = $task.taskId
    Write-Host "  Task created: $taskId" -ForegroundColor Green
} catch {
    Write-Host "  Create Task FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Process Task
Write-Host "[3/6] Process Task..." -ForegroundColor Yellow
try {
    $process = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/process" -Method Post -Headers $headers -TimeoutSec 30
    Write-Host "  Process SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Process Task FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Set all required fields
Write-Host "[4/6] Set all required fields..." -ForegroundColor Yellow
$fields = @(
    @{ fieldName = "fornecedor"; value = "Dell Technologies" },
    @{ fieldName = "valor"; value = "R$ 50.000,00" },
    @{ fieldName = "justificativa"; value = "Renovação de equipamentos" },
    @{ fieldName = "categoria_produto"; value = "Notebooks - TI" },
    @{ fieldName = "data_solicitacao"; value = "2026-01-15" },
    @{ fieldName = "solicitante"; value = "João Silva" },
    @{ fieldName = "area_solicitante"; value = "TI" },
    @{ fieldName = "aprovacao_cade"; value = "Não se aplica" }
)
foreach ($f in $fields) {
    try {
        $dataBody = $f | ConvertTo-Json
        $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    } catch {
        Write-Host "  Set Data FAILED for $($f.fieldName): $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
}
Write-Host "  All fields set" -ForegroundColor Green

# Validate
Write-Host "[5/6] Validate..." -ForegroundColor Yellow
try {
    $validate = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/validate" -Method Post -Headers $headers -TimeoutSec 15
    Write-Host "  Validate SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Validate FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Generate Document - Document Engine is stopped, should fallback to Go
Write-Host "[6/6] Generate Document (fallback to Go)..." -ForegroundColor Yellow
try {
    $generate = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/generate" -Method Post -Headers $headers -TimeoutSec 60
    Write-Host "  Generate SUCCESS (fallback worked!)" -ForegroundColor Green
    Write-Host "  DocumentRunId: $($generate.documentRunId)" -ForegroundColor Gray
    Write-Host "  DocxPath: $($generate.docxPath)" -ForegroundColor Gray
    Write-Host "  Status: $($generate.status)" -ForegroundColor Gray
    $runId = $generate.documentRunId
} catch {
    Write-Host "  Generate FAILED: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Response: $($_.ErrorDetails.Message)" -ForegroundColor Red
    exit 1
}

# Download DOCX
Write-Host "[7/7] Download DOCX..." -ForegroundColor Yellow
try {
    $docxResponse = Invoke-WebRequest -Uri "$baseUrl/api/documents/$runId/docx" -Headers $headers -Method Get -TimeoutSec 30 -OutFile "test_fallback.docx"
    $fileSize = (Get-Item "test_fallback.docx").Length
    Write-Host "  DOCX downloaded: $fileSize bytes" -ForegroundColor Green
} catch {
    Write-Host "  Download DOCX FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== E2E #6 Document Engine Fallback Test PASSED ===" -ForegroundColor Green