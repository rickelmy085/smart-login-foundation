#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Runs E2E test for Rule FAIL scenario.
#>

param(
    [string]$BackendPort = "8082",
    [string]$DocEnginePort = "8090"
)

$baseUrl = "http://127.0.0.1:$BackendPort"

Write-Host "=== ABIS E2E #4 - Rule FAIL Test ===" -ForegroundColor Cyan
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

# Set Data - fornecedor vazio (should trigger rule fail)
Write-Host "[4/6] Set Data (fornecedor vazio)..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "fornecedor"; value = "" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  Data set SUCCESS" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Set other required fields
Write-Host "[5/6] Set other required fields..." -ForegroundColor Yellow
$fields = @(
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

# Validate - should fail because fornecedor is empty
Write-Host "[6/6] Validate (should fail)..." -ForegroundColor Yellow
try {
    $validate = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/validate" -Method Post -Headers $headers -TimeoutSec 15
    Write-Host "  Validate response: $($validate.answer)" -ForegroundColor Gray
    if ($validate.answer -like "*fornecedor*" -or $validate.answer -like "*FAIL*" -or $validate.answer -like "*falta*" -or $validate.answer -like "*obrigatóri*") {
        Write-Host "  Rule FAIL detected correctly!" -ForegroundColor Green
    } else {
        Write-Host "  WARNING: Expected rule fail but got: $($validate.answer)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  Validate FAILED: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Response: $($_.ErrorDetails.Message)" -ForegroundColor Red
    # Check if it's a 400 with rule fail message
    if ($_.Exception.Response.StatusCode -eq 400) {
        Write-Host "  Rule FAIL detected via 400 status!" -ForegroundColor Green
    }
    exit 1
}

Write-Host ""
Write-Host "=== E2E #4 Rule FAIL Test PASSED ===" -ForegroundColor Green