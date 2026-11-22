# Activates the pre-commit hook (.githooks/pre-commit, runs scripts/cicheck).
# No bash/make required; Git for Windows runs the hook with its bundled sh.
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

git config core.hooksPath .githooks
# Remove a stale binary hook from older versions (Git for Windows cannot spawn it).
Remove-Item -Force -ErrorAction SilentlyContinue .githooks/pre-commit.exe

Write-Host "Pre-commit hook installed: .githooks/pre-commit"
