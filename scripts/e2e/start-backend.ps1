#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Starts the Go Backend server.
.DESCRIPTION
    Runs the Go HTTP server with all configured middleware and routes.
#>

param(
    [string]$Port = "8081"
)

Set-Location (Split-Path -Parent $PSScriptRoot) -Parent
Set-Location "backend"

Write-Host "Starting Go Backend on port $Port..." -ForegroundColor Cyan

$env:PORT = $Port
.\abis.exe