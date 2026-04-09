#!/usr/bin/env pwsh
# cleanup-axiom.ps1
# Place ce script à la racine du projet Axiom et lance-le une fois.
# Il supprime les fichiers superflus et met en place les fichiers propres.

$ROOT = $PSScriptRoot

Write-Host "=== Axiom Cleanup ===" -ForegroundColor Cyan

# ── 1. Supprimer les fichiers de docs v0.3 superflus ─────────────
$junk = @(
    "AXIOM_v0.3_COMPONENTS_GUIDE.md",
    "AXIOM_v0.3_IMPROVEMENTS.md",
    "CHANGELOG_v0.3.md",
    "INDEX_AXIOM_v0.3.md",
    "QUICK_START.md",
    "installation.md",
    "FINAL_SUMMARY.md",
    "AXIOM_DETAILED_ANALYSIS.md",
    "AXIOM_LAUNCH_GUIDE.md",
    "install-axiom.ps1",
    "fix-wails-json.ps1",
    "frontend\index-v0.3.html",
    "api\events_extended.go"   # fusionné dans api/events.go
)

foreach ($f in $junk) {
    $path = Join-Path $ROOT $f
    if (Test-Path $path) {
        Remove-Item $path -Force
        Write-Host "  supprimé : $f" -ForegroundColor Yellow
    }
}

# ── 2. Vider demo/ si présent ─────────────────────────────────────
$demoDir = Join-Path $ROOT "demo"
if (Test-Path $demoDir) {
    Remove-Item $demoDir -Recurse -Force
    Write-Host "  supprimé : demo/" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "  Remplace maintenant les fichiers suivants par les versions propres :"
Write-Host "  - api/events.go           (fusionné avec events_extended.go)"
Write-Host "  - main.go                 (nettoyé, sans démo verbose)"
Write-Host "  - frontend/index.html     (shell VSCode-like prêt)"
Write-Host "  - ARCHITECTURE.md         (doc courte et factuelle)"
Write-Host ""
Write-Host "=== Terminé ===" -ForegroundColor Green
