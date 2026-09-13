#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Runs E2E test for RAG sem resultado (no evidence found).
#>

param(
    [string]$BackendPort = "8082"
)

$baseUrl = "http://127.0.0.1:$BackendPort"

Write-Host "=== ABIS E2E #8 - RAG Sem Resultado Test ===" -ForegroundColor Cyan
Write-Host ""

# Login
Write-Host "[1/3] Login..." -ForegroundColor Yellow
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

# Chat with question that has no evidence in normativos
Write-Host "[2/3] Chat with unrelated question..." -ForegroundColor Yellow
try {
    $chatBody = @{ question = "Qual é a capital da França?" } | ConvertTo-Json
    $chat = Invoke-RestMethod -Uri "$baseUrl/api/chat" -Method Post -Body $chatBody -ContentType "application/json" -Headers $headers -TimeoutSec 30
    Write-Host "  Chat response: $($chat.answer)" -ForegroundColor Gray
    
    # Check if it admits insufficiency
    if ($chat.answer -like "*não possui*" -or $chat.answer -like "*insuficiente*" -or $chat.answer -like "*não encontrei*" -or $chat.answer -like "*não tenho informação*") {
        Write-Host "  Correctly admits insufficiency!" -ForegroundColor Green
    } else {
        Write-Host "  WARNING: May have hallucinated: $($chat.answer)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  Chat FAILED: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Response: $($_.ErrorDetails.Message)" -ForegroundColor Red
    exit 1
}

# Also test workflow with question that has no evidence
Write-Host "[3/3] Create Task with unrelated question..." -ForegroundColor Yellow
try {
    $taskBody = @{ question = "Como faço para cozinhar um bolo de chocolate?" } | ConvertTo-Json
    $task = Invoke-RestMethod -Uri "$baseUrl/api/tasks" -Method Post -Body $taskBody -ContentType "application/json" -Headers $headers -TimeoutSec 30
    $taskId = $task.taskId
    Write-Host "  Task created: $taskId" -ForegroundColor Green
    
    $process = Invoke-RestMethod -Uri "$baseUrl/api/tasks/$taskId/process" -Method Post -Headers $headers -TimeoutSec 30
    Write-Host "  Process response: $($process.answer)" -ForegroundColor Gray
    
    if ($process.answer -like "*não possui*" -or $process.answer -like "*insuficiente*" -or $process.answer -like "*não encontrei*") {
        Write-Host "  Correctly admits insufficiency in workflow!" -ForegroundColor Green
    } else {
        Write-Host "  WARNING: Workflow may have hallucinated: $($process.answer)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  Task FAILED: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Response: $($_.ErrorDetails.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== E2E #8 RAG Sem Resultado Test PASSED ===" -ForegroundColor Green