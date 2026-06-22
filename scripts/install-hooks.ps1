# Installs the pre-commit hook on Windows (no bash/make required).
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

git config core.hooksPath .githooks
go build -o .githooks/pre-commit.exe ./scripts/cicheck

Write-Host "Pre-commit hook installed: .githooks/pre-commit.exe"
