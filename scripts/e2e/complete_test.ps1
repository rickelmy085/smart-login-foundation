#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Runs the complete E2E test for ABIS including document generation.
#>

param(
    [string]$BackendPort = "8082",
    [string]$DocEnginePort = "8090"
)

$baseUrl = "http://127.0.0.1:$BackendPort"
$docEngineUrl = "http://127.0.0.1:$DocEnginePort"

Write-Host "=== ABIS E2E Complete Test ===" -ForegroundColor Cyan
Write-Host "Backend: $baseUrl" -ForegroundColor Gray
Write-Host "Document Engine: $docEngineUrl" -ForegroundColor Gray
Write-Host ""

# Test 1: Health check
Write-Host "[1/12] Health Check..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$baseUrl/api/health" -Method Get -TimeoutSec 10
    Write-Host "  Backend health: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "  Backend health FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

try {
    $docHealth = Invoke-RestMethod -Uri "$docEngineUrl/health" -Method Get -TimeoutSec 10
    Write-Host "  Document Engine health: $($docHealth.status)" -ForegroundColor Green
    Write-Host "  Templates loaded: $($docHealth.templates_loaded)" -ForegroundColor Gray
} catch {
    Write-Host "  Document Engine health FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 2: Metrics
Write-Host "[2/12] Metrics..." -ForegroundColor Yellow
try {
    $metrics = Invoke-RestMethod -Uri "$baseUrl/api/metrics" -Method Get -TimeoutSec 10
    Write-Host "  Metrics available: $($metrics.Count) keys" -ForegroundColor Green
} catch {
    Write-Host "  Metrics FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 3: Login
Write-Host "[3/12] Login..." -ForegroundColor Yellow
try {
    $loginBody = @{ re = "123456"; password = "demo123"; remember = $false } | ConvertTo-Json
    $login = Invoke-RestMethod -Uri "$baseUrl/api/login" -Method Post -Body $loginBody -ContentType "application/json" -TimeoutSec 15
    $token = $login.token
    $employeeId = $login.employee.id
    Write-Host "  Login SUCCESS: $($login.employee.name)" -ForegroundColor Green
} catch {
    Write-Host "  Login FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

$headers = @{ Authorization = "Bearer $token" }

# Test 4: Create Task
Write-Host "[4/12] Create Task..." -ForegroundColor Yellow
try {
    $taskBody = @{ question = "Preciso solicitar aquisição de notebooks Dell para a agência, valor R$ 50.000, justificativa renovação de equipamentos" } | ConvertTo-Json
    $task = Invoke-RestMethod -Uri "$baseUrl/api/tasks" -Method Post -Body $taskBody -ContentType "application/json" -Headers $headers -TimeoutSec 30
    $taskId = $task.taskId
    Write-Host "  Task created: $taskId" -ForegroundColor Green
} catch {
    Write-Host "  Create Task FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 5: Process Task
Write-Host "[5/12] Process Task..." -ForegroundColor Yellow
try {
    $process = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/process" -Method Post -Headers $headers -TimeoutSec 30
    Write-Host "  Process SUCCESS" -ForegroundColor Green
    Write-Host "  Requirements: $($process.requirements.Count)" -ForegroundColor Gray
} catch {
    Write-Host "  Process Task FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 6: Set Data - fornecedor
Write-Host "[6/12] Set Data (fornecedor)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "fornecedor"; value = "Dell Technologies" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  Data set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 7: Set Data - valor
Write-Host "[7/12] Set Data (valor)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "valor"; value = "R$ 50.000,00" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  Data set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 8: Set Data - justificativa
Write-Host "[8/12] Set Data (justificativa)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "justificativa"; value = "Renovação de equipamentos obsoletos da agência" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  Data set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 9: Set Data - categoria_produto
Write-Host "[9/12] Set Data (categoria_produto)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "categoria_produto"; value = "Notebooks - TI" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  Data set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 10: Set Data - data_solicitacao
Write-Host "[10/12] Set Data (data_solicitacao)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "data_solicitacao"; value = "2026-01-15" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  Data set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 11: Set Data - solicitante and area_solicitante
Write-Host "[11/14] Set Data (solicitante, area_solicitante)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "solicitante"; value = "João Silva" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  solicitante set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

try {
    $dataBody = @{ fieldName = "area_solicitante"; value = "TI" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  area_solicitante set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 12: Set Data (aprovacao_cade)
Write-Host "[12/14] Set Data (aprovacao_cade)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "aprovacao_cade"; value = "Não se aplica" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  aprovacao_cade set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 13: Validate
Write-Host "[13/14] Validate..." -ForegroundColor Yellow
try {
    $validate = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/validate" -Method Post -Headers $headers -TimeoutSec 15
    Write-Host "  Validate SUCCESS" -ForegroundColor Green
    Write-Host "  Answer: $($validate.answer)" -ForegroundColor Gray
    Write-Host "  ReadyToGenerate: $($validate.readyToGen)" -ForegroundColor Gray
} catch {
    Write-Host "  Validate FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 14: Generate Document
Write-Host "[14/14] Generate Document..." -ForegroundColor Yellow
try {
    $generate = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/generate" -Method Post -Headers $headers -TimeoutSec 60
    Write-Host "  Generate SUCCESS" -ForegroundColor Green
    Write-Host "  DocumentRunId: $($generate.documentRunId)" -ForegroundColor Gray
    Write-Host "  DocxPath: $($generate.docxPath)" -ForegroundColor Gray
    Write-Host "  Status: $($generate.status)" -ForegroundColor Gray
    $runId = $generate.documentRunId
} catch {
    Write-Host "  Generate FAILED: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Response: $($_.ErrorDetails.Message)" -ForegroundColor Red
    exit 1
}

# Test 14: Download DOCX
Write-Host "[14/14] Download DOCX..." -ForegroundColor Yellow
try {
    $docxResponse = Invoke-WebRequest -Uri "$baseUrl/api/documents/$runId/docx" -Headers $headers -Method Get -TimeoutSec 30 -OutFile "test_output.docx"
    $fileSize = (Get-Item "test_output.docx").Length
    Write-Host "  DOCX downloaded: $fileSize bytes" -ForegroundColor Green
} catch {
    Write-Host "  Download DOCX FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== E2E Complete Test PASSED ===" -ForegroundColor Green
Write-Host "Task ID: $taskId" -ForegroundColor Gray
Write-Host "Document Run ID: $runId" -ForegroundColor Gray
Write-Host "DOCX file: test_output.docx" -ForegroundColor Gray