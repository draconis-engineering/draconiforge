# Draconiforge installer — PowerShell
# One-liner: irm https://raw.githubusercontent.com/draconis-engineering/draconiforge/main/scripts/install.ps1 | iex
param([string]$Prefix = "", [string]$Version = "main", [switch]$NoModifyPath)
$ErrorActionPreference = "Stop"
$Repo = "https://github.com/draconis-engineering/draconiforge.git"
$Bin = "draconiforge"; $Alias = "forge"
function Info($m){ Write-Host "==> $m" -ForegroundColor Blue }
function Ok($m){ Write-Host "✓ $m" -ForegroundColor Green }
function Warn($m){ Write-Host "⚠ $m" -ForegroundColor Yellow }
function Fail($m){ Write-Host "✖ $m" -ForegroundColor Red; exit 1 }
if (-not $Prefix) { if ($env:PREFIX) { $Prefix = $env:PREFIX } elseif ($env:LOCALAPPDATA) { $Prefix = Join-Path $env:LOCALAPPDATA "draconiforge" } else { $Prefix = Join-Path $env:USERPROFILE ".local" } }
$Bindir = Join-Path $Prefix "bin"; $Datadir = Join-Path $Prefix "share\draconiforge\scripts"
if (-not (Get-Command git -ErrorAction SilentlyContinue)) { Fail "git not found" }
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { Fail "go not found — install Go 1.24+ from https://go.dev/dl/" }
Info "Installing $Bin ($Alias) from $Repo@$Version"
$Tmp = Join-Path $env:TEMP ("draconiforge-" + [Guid]::NewGuid().ToString().Substring(0,8))
New-Item -ItemType Directory -Path $Tmp | Out-Null; $Src = Join-Path $Tmp "src"
try {
  Info "Cloning $Repo ..."
  & git clone --depth 1 --branch $Version $Repo $Src 2>$null; if ($LASTEXITCODE -ne 0) { & git clone --depth 1 $Repo $Src }
  if ($LASTEXITCODE -ne 0) { Fail "git clone failed" }
  Info "Building $Bin ..."
  Push-Location $Src
  if ((Get-Command make -ErrorAction SilentlyContinue) -and (Test-Path "Makefile")) { & make build 2>&1 | ForEach-Object { "  $_" }; $BinSrc = Join-Path $Src "draconiforge.exe"; if (-not (Test-Path $BinSrc)) { $BinSrc = Join-Path $Src "draconiforge" } }
  else { & go build -ldflags "-s -w" -o (Join-Path $Tmp "$Bin.exe") ./cmd/goforge-cli; if ($LASTEXITCODE -ne 0) { Fail "go build failed" }; $BinSrc = Join-Path $Tmp "$Bin.exe" }
  Pop-Location
  if (-not (Test-Path $BinSrc)) { $BinSrc = Join-Path $Src $Bin; if (-not (Test-Path $BinSrc)) { Fail "build failed" } }
  New-Item -ItemType Directory -Force -Path $Bindir | Out-Null; New-Item -ItemType Directory -Force -Path $Datadir | Out-Null
  Copy-Item -Force $BinSrc (Join-Path $Bindir "$Bin.exe"); Copy-Item -Force (Join-Path $Bindir "$Bin.exe") (Join-Path $Bindir "$Alias.exe")
  Ok "installed $Bindir\$Bin.exe + $Alias.exe"
  Get-ChildItem (Join-Path $Src "scripts") -Filter *.sh -ErrorAction SilentlyContinue | Where-Object { $_.Name -notin @("install.sh","install.ps1") } | ForEach-Object { Copy-Item $_.FullName $Datadir -Force }
  Get-ChildItem (Join-Path $Src "scripts") -Filter *.ps1 -ErrorAction SilentlyContinue | Where-Object { $_.Name -notin @("install.sh","install.ps1") } | ForEach-Object { Copy-Item $_.FullName $Datadir -Force }
  Ok "scripts → $Datadir"
  try { $v = & (Join-Path $Bindir "$Bin.exe") version 2>$null; if ($v) { Ok "version: $v" } } catch {}
  $UserPath = [Environment]::GetEnvironmentVariable("Path","User")
  if ($UserPath -split ";" -notcontains $Bindir) {
    Warn "$Bindir not in PATH"
    if (-not $NoModifyPath) { [Environment]::SetEnvironmentVariable("Path", "$UserPath;$Bindir", "User"); $env:Path += ";$Bindir"; Ok "added to user PATH — restart terminal" }
  } else { Ok "$Bindir already in PATH" }
  Write-Host ""; Ok "Try: forge doctor --verbose"
} finally { Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue }
