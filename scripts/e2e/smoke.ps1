#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Runs the complete E2E smoke test for ABIS.
.DESCRIPTION
    Executes a complete end-to-end test: login, chat, workflow, document generation, and validation.
#>

param(
    [string]$BackendPort = "8082",
    [string]$DocEnginePort = "8090"
)

$baseUrl = "http://127.0.0.1:$BackendPort"
$docEngineUrl = "http://127.0.0.1:$DocEnginePort"

Write-Host "=== ABIS E2E Smoke Test ===" -ForegroundColor Cyan
Write-Host "Backend: $baseUrl" -ForegroundColor Gray
Write-Host "Document Engine: $docEngineUrl" -ForegroundColor Gray
Write-Host ""

# Test 1: Health check
Write-Host "[1/8] Health Check..." -ForegroundColor Yellow
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
    Write-Host "  Template keys: $($docHealth.template_keys -join ', ')" -ForegroundColor Gray
} catch {
    Write-Host "  Document Engine health FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 2: Metrics
Write-Host "[2/8] Metrics..." -ForegroundColor Yellow
try {
    $metrics = Invoke-RestMethod -Uri "$baseUrl/api/metrics" -Method Get -TimeoutSec 10
    Write-Host "  Metrics available: $($metrics.Count) keys" -ForegroundColor Green
} catch {
    Write-Host "  Metrics FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 3: Login
Write-Host "[3/8] Login..." -ForegroundColor Yellow
try {
    $loginBody = @{ re = "123456"; password = "demo123"; remember = $false } | ConvertTo-Json
    $login = Invoke-RestMethod -Uri "$baseUrl/api/login" -Method Post -Body $loginBody -ContentType "application/json" -TimeoutSec 15
    $token = $login.token
    $employeeId = $login.employee.id
    Write-Host "  Login SUCCESS: $($login.employee.name)" -ForegroundColor Green
    Write-Host "  Token length: $($token.Length)" -ForegroundColor Gray
} catch {
    Write-Host "  Login FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 4: Chat with RAG
Write-Host "[4/8] Chat/RAG..." -ForegroundColor Yellow
try {
    $headers = @{ Authorization = "Bearer $token" }
    $chatBody = @{ question = "Qual é a política de compras da organização?" } | ConvertTo-Json
    $chat = Invoke-RestMethod -Uri "$baseUrl/api/chat" -Method Post -Body $chatBody -ContentType "application/json" -Headers $headers -TimeoutSec 30
    Write-Host "  Chat SUCCESS" -ForegroundColor Green
    Write-Host "  Answer: $($chat.answer.Substring(0, [Math]::Min(100, $chat.answer.Length)))..." -ForegroundColor Gray
    Write-Host "  Sources: $($chat.sources.Count)" -ForegroundColor Gray
} catch {
    Write-Host "  Chat FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 5: Create Task
Write-Host "[5/8] Create Task..." -ForegroundColor Yellow
try {
    $taskBody = @{ question = "Preciso solicitar aquisição de notebooks para a agência" } | ConvertTo-Json
    $task = Invoke-RestMethod -Uri "$baseUrl/api/tasks" -Method Post -Body $taskBody -ContentType "application/json" -Headers $headers -TimeoutSec 30
    $taskId = $task.taskId
    Write-Host "  Task created: $taskId" -ForegroundColor Green
    Write-Host "  Status: $($task.status)" -ForegroundColor Gray
} catch {
    Write-Host "  Create Task FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 6: Process Task
Write-Host "[6/8] Process Task..." -ForegroundColor Yellow
try {
    $process = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/process" -Method Post -Headers $headers -TimeoutSec 30
    Write-Host "  Process SUCCESS" -ForegroundColor Green
    Write-Host "  Requirements: $($process.requirements.Count)" -ForegroundColor Gray
} catch {
    Write-Host "  Process Task FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 7: Set Data
Write-Host "[7/8] Set Data..." -ForegroundColor Yellow
try {
    $dataBody = @{ fieldName = "fornecedor"; value = "Dell Technologies" } | ConvertTo-Json
    $setData = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/data" -Method Post -Body $dataBody -ContentType "application/json" -Headers $headers -TimeoutSec 15
    Write-Host "  Data set SUCCESS: $($setData.status)" -ForegroundColor Green
} catch {
    Write-Host "  Set Data FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test 8: Validate
Write-Host "[8/8] Validate..." -ForegroundColor Yellow
try {
    $validate = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/validate" -Method Post -Headers $headers -TimeoutSec 15
    Write-Host "  Validate SUCCESS" -ForegroundColor Green
    Write-Host "  Answer: $($validate.answer)" -ForegroundColor Gray
} catch {
    Write-Host "  Validate FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== E2E Smoke Test Complete ===" -ForegroundColor Green
Write-Host "Task ID: $taskId" -ForegroundColor Gray
Write-Host "Employee ID: $employeeId" -ForegroundColor Gray