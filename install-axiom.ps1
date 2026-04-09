$ErrorActionPreference = "Stop"

function Write-Ok   { param($msg) Write-Host "  [OK] $msg" -ForegroundColor Green }
function Write-Info { param($msg) Write-Host "  [..] $msg" -ForegroundColor Cyan }
function Write-Warn { param($msg) Write-Host "  [!!] $msg" -ForegroundColor Yellow }
function Write-Fail { param($msg) Write-Host "  [XX] $msg" -ForegroundColor Red }
function Write-Step { param($msg) Write-Host "`n=== $msg ===" -ForegroundColor Magenta }

Clear-Host
Write-Host "  AXIOM IDE - Installateur Windows v0.3.0" -ForegroundColor Blue
Write-Host ""

# =====================================================
# CONFIGURATION - Modifier si besoin
# =====================================================
# Repertoire du projet Axiom (votre code local)
$PROJECT_DIR    = $PSScriptRoot
$GO_VERSION_MIN = [Version]"1.22.0"
$GO_DOWNLOAD    = "https://go.dev/dl/go1.22.5.windows-amd64.msi"
$WAILS_PKG      = "github.com/wailsapp/wails/v2/cmd/wails@latest"
$WEBVIEW2_URL   = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"

Write-Host "  Projet detecte : $PROJECT_DIR" -ForegroundColor Cyan
Write-Host ""

$script:errs = @()

# =====================================================
# ETAPE 1 - Verification systeme
# =====================================================
Write-Step "Etape 1 - Verification systeme"

$winVer = [System.Environment]::OSVersion.Version
if ($winVer.Major -lt 10) {
    Write-Fail "Windows 10 ou superieur requis"
    $script:errs += "Windows trop ancien"
} else {
    Write-Ok "Windows $($winVer.Major).$($winVer.Minor) detecte"
}

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")
if (-not $isAdmin) {
    Write-Warn "Non lance en administrateur - certaines etapes peuvent echouer"
} else {
    Write-Ok "Droits administrateur confirmes"
}

# Verifier que go.mod existe dans le dossier
if (-not (Test-Path "$PROJECT_DIR\go.mod")) {
    Write-Fail "go.mod introuvable dans $PROJECT_DIR"
    Write-Fail "Placez ce script dans le dossier racine du projet Axiom"
    $script:errs += "go.mod introuvable"
    Read-Host "Appuyer sur Entree pour fermer"
    exit 1
}
Write-Ok "go.mod trouve - projet Axiom valide"

# =====================================================
# ETAPE 2 - Go
# =====================================================
Write-Step "Etape 2 - Go (minimum 1.22)"

$goOk = $false
try {
    $goRaw = "$(& go version 2>&1)"
    $goRaw = $goRaw -replace "go version go", ""
    $goRaw = $goRaw -replace " windows.*", ""
    $goVer = [Version]($goRaw.Trim().Split(" ")[0])
    if ($goVer -ge $GO_VERSION_MIN) {
        Write-Ok "Go $goVer detecte et compatible"
        $goOk = $true
    } else {
        Write-Warn "Go $goVer trop ancien - mise a jour necessaire"
    }
} catch {
    Write-Warn "Go non trouve"
}

if (-not $goOk) {
    Write-Info "Telechargement de Go 1.22.5..."
    $msiPath = "$env:TEMP\go-installer.msi"
    try {
        Invoke-WebRequest -Uri $GO_DOWNLOAD -OutFile $msiPath -UseBasicParsing
        Write-Info "Installation Go en cours (1-2 min)..."
        Start-Process msiexec.exe -ArgumentList "/i `"$msiPath`" /quiet /norestart" -Wait
        $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("PATH","User")
        Write-Ok "Go installe - redemarrez le terminal apres l installation"
        Remove-Item $msiPath -Force -ErrorAction SilentlyContinue
    } catch {
        Write-Fail "Echec installation Go: $_"
        $script:errs += "Go non installe"
    }
}

# =====================================================
# ETAPE 3 - Wails
# =====================================================
Write-Step "Etape 3 - Wails v2"

$wailsOk = $false
try {
    $null = & wails version 2>&1
    Write-Ok "Wails deja installe"
    $wailsOk = $true
} catch {
    Write-Info "Installation de Wails..."
}

if (-not $wailsOk) {
    try {
        & go install $WAILS_PKG
        $gobin = "$env:USERPROFILE\go\bin"
        if ($env:PATH -notlike "*$gobin*") {
            $env:PATH += ";$gobin"
        }
        Write-Ok "Wails installe"
    } catch {
        Write-Warn "Echec Wails - mode console reste disponible sans lui"
    }
}

# =====================================================
# ETAPE 4 - WebView2
# =====================================================
Write-Step "Etape 4 - WebView2 Runtime"

$wv2Key = "HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
if (Test-Path $wv2Key) {
    Write-Ok "WebView2 deja installe"
} else {
    Write-Info "Telechargement WebView2..."
    $wv2Path = "$env:TEMP\WebView2Setup.exe"
    try {
        Invoke-WebRequest -Uri $WEBVIEW2_URL -OutFile $wv2Path -UseBasicParsing
        Start-Process $wv2Path -ArgumentList "/silent /install" -Wait
        Write-Ok "WebView2 installe"
        Remove-Item $wv2Path -Force -ErrorAction SilentlyContinue
    } catch {
        Write-Warn "Echec WebView2 - requis pour l interface graphique"
    }
}

# =====================================================
# ETAPE 5 - GCC
# =====================================================
Write-Step "Etape 5 - GCC (compilateur C pour Wails)"

$gccOk = $false
try {
    $null = & gcc --version 2>&1
    Write-Ok "GCC deja present"
    $gccOk = $true
} catch {
    Write-Info "GCC non trouve - telechargement TDM-GCC..."
}

if (-not $gccOk) {
    $tdmUrl  = "https://github.com/jmeubank/tdm-gcc/releases/download/v10.3.0-tdm64-2/tdm64-gcc-10.3.0-2.exe"
    $tdmPath = "$env:TEMP\tdm-gcc.exe"
    try {
        Invoke-WebRequest -Uri $tdmUrl -OutFile $tdmPath -UseBasicParsing
        Write-Info "Installation TDM-GCC (peut prendre quelques minutes)..."
        Start-Process $tdmPath -ArgumentList "/S" -Wait
        $env:PATH += ";C:\TDM-GCC-64\bin"
        Write-Ok "TDM-GCC installe"
        Remove-Item $tdmPath -Force -ErrorAction SilentlyContinue
    } catch {
        Write-Warn "Echec GCC - installer depuis https://jmeubank.github.io/tdm-gcc/"
        Write-Warn "Requis uniquement pour wails build/dev"
    }
}

# =====================================================
# ETAPE 6 - Dependances Go
# =====================================================
Write-Step "Etape 6 - Dependances Go"

try {
    Push-Location $PROJECT_DIR
    Write-Info "go mod download..."
    & go mod download
    Write-Info "go mod tidy..."
    & go mod tidy
    Pop-Location
    Write-Ok "Dependances installees"
} catch {
    Write-Fail "Erreur dependances: $_"
    $script:errs += "Dependances Go echouees"
    Pop-Location
}

# =====================================================
# ETAPE 7 - Creation wails.json (requis par Wails)
# =====================================================
Write-Step "Etape 7 - Configuration Wails (wails.json)"

$wailsJson = "$PROJECT_DIR\wails.json"

if (Test-Path $wailsJson) {
    Write-Ok "wails.json deja present"
} else {
    Write-Info "Creation de wails.json..."

    # Lire le nom du module depuis go.mod
    $moduleName = "axiom"
    try {
        $goModContent = Get-Content "$PROJECT_DIR\go.mod" -Raw
        if ($goModContent -match "^module\s+(.+)$") {
            $moduleName = $Matches[1].Split("/")[-1].Trim()
        }
    } catch {}

    # Creer le dossier frontend si absent
    $frontendDir = "$PROJECT_DIR\frontend"
    if (-not (Test-Path $frontendDir)) {
        New-Item -ItemType Directory -Path $frontendDir -Force | Out-Null
        Write-Info "Dossier frontend/ cree"
    }

    # Creer index.html minimal si absent
    $indexHtml = "$frontendDir\index.html"
    if (-not (Test-Path $indexHtml)) {
        $htmlContent = @"
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Axiom IDE</title>
    <link rel="stylesheet" href="./axiom-ui-components.css">
    <style>
        body { margin: 0; font-family: sans-serif; background: #1e1e1e; color: #d4d4d4; }
        .center { display: flex; align-items: center; justify-content: center; height: 100vh; flex-direction: column; gap: 16px; }
        h1 { color: #007acc; }
    </style>
</head>
<body>
    <div class="center">
        <h1>Axiom IDE v0.3.0</h1>
        <p>Moteur charge - interface en cours de developpement</p>
    </div>
    <script src="./axiom-ui-components.js" defer></script>
</body>
</html>
"@
        Set-Content -Path $indexHtml -Value $htmlContent -Encoding UTF8
        Write-Info "frontend/index.html cree"
    }

    # Contenu wails.json
    $wailsContent = @"
{
  "name": "Axiom IDE",
  "outputfilename": "axiom",
  "frontend:install": "echo no-install",
  "frontend:build": "echo no-build",
  "frontend:dev:watcher": "",
  "frontend:dev:serverUrl": "",
  "wailsjsdir": "./frontend",
  "version": "2",
  "info": {
    "companyName": "Axiom IDE",
    "productName": "Axiom IDE",
    "productVersion": "0.3.0",
    "copyright": "MIT",
    "comments": "Modular Software Engineering Platform"
  },
  "assetdir": "frontend",
  "reloaddirs": "",
  "build": {
    "appargs": ""
  },
  "platforms": [
    {
      "name": "windows",
      "windowsConfig": {
        "webviewIsTransparent": false,
        "windowIsTranslucent": false,
        "disableWindowIcon": false
      }
    }
  ],
  "bindings": {
    "ts_generation": {
      "prefix": "",
      "suffix": ""
    }
  }
}
"@
    Set-Content -Path $wailsJson -Value $wailsContent -Encoding UTF8
    Write-Ok "wails.json cree"
}

# =====================================================
# ETAPE 8 - Compilation console (go build)
# =====================================================
Write-Step "Etape 8 - Compilation mode console"

try {
    Push-Location $PROJECT_DIR
    Write-Info "go build -o axiom.exe ./..."
    & go build -o axiom.exe ./
    Pop-Location
    Write-Ok "axiom.exe compile avec succes"
} catch {
    Write-Fail "Echec compilation: $_"
    $script:errs += "Compilation echouee"
    Pop-Location
}

# =====================================================
# ETAPE 9 - Raccourci Bureau
# =====================================================
Write-Step "Etape 9 - Raccourci Bureau"

try {
    $exePath  = "$PROJECT_DIR\axiom.exe"
    $lnkPath  = "$env:USERPROFILE\Desktop\Axiom IDE.lnk"
    $shell    = New-Object -ComObject WScript.Shell
    $sc       = $shell.CreateShortcut($lnkPath)
    $sc.TargetPath       = $exePath
    $sc.WorkingDirectory = $PROJECT_DIR
    $sc.Description      = "Axiom IDE v0.3.0"
    $sc.Save()
    Write-Ok "Raccourci cree sur le Bureau"
} catch {
    Write-Warn "Impossible de creer le raccourci: $_"
}

# =====================================================
# RAPPORT FINAL
# =====================================================
Write-Host ""
Write-Host "============================================" -ForegroundColor Magenta
Write-Host "  INSTALLATION TERMINEE" -ForegroundColor Magenta
Write-Host "============================================" -ForegroundColor Magenta
Write-Host ""

if ($script:errs.Count -eq 0) {
    Write-Host "  Tout installe sans erreur !" -ForegroundColor Green
} else {
    Write-Host "  $($script:errs.Count) probleme(s) :" -ForegroundColor Yellow
    foreach ($e in $script:errs) {
        Write-Host "    - $e" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "  Projet : $PROJECT_DIR" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Pour lancer Axiom :" -ForegroundColor Cyan
Write-Host ""
Write-Host "    [Console uniquement]" -ForegroundColor Gray
Write-Host "    .\axiom.exe" -ForegroundColor White
Write-Host ""
Write-Host "    [Interface graphique - dev avec hot reload]" -ForegroundColor Gray
Write-Host "    wails dev" -ForegroundColor White
Write-Host ""
Write-Host "    [Interface graphique - build final .exe]" -ForegroundColor Gray
Write-Host "    wails build -platform windows/amd64" -ForegroundColor White
Write-Host ""

Read-Host "Appuyer sur Entree pour fermer"