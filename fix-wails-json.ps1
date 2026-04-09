$PROJECT_DIR = $PSScriptRoot
$wailsJson   = "$PROJECT_DIR\wails.json"

$content = @"
{
  "name": "Axiom IDE",
  "outputfilename": "axiom",
  "frontend:install": "echo no-install",
  "frontend:build": "echo no-build",
  "wailsjsdir": "./frontend",
  "version": "2",
  "info": {
    "companyName": "Axiom IDE",
    "productName": "Axiom IDE",
    "productVersion": "0.3.0",
    "copyright": "MIT"
  },
  "assetdir": "frontend"
}
"@

# Ecrire sans BOM (ASCII/UTF8 sans signature)
$encoding = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($wailsJson, $content, $encoding)

Write-Host "wails.json corrige (sans BOM) -> $wailsJson" -ForegroundColor Green
Write-Host "Lancez maintenant: wails dev" -ForegroundColor Cyan
