<#
.SYNOPSIS
  Builds XinText for Windows and produces a portable zip, optionally
  bundling pandoc and an NSIS installer.

.DESCRIPTION
  Portable layout (what the Go backend auto-detects):
      XinText/
        XinText.exe
        pandoc/pandoc.exe        (optional, used for HTML/DOCX/TXT export)

  PDF export needs no external engine: it is rendered by the system
  Microsoft Edge (headless --print-to-pdf).

  Resolution in the app: XinText_PANDOC env -> bundled pandoc -> PATH.

.PARAMETER Pandoc
  Path to the pandoc.exe to bundle. Defaults to $env:PANDOC_BIN, then a
  known development location. Skip bundling with -SkipPandoc.

.PARAMETER SkipBuild
  Reuse the existing bin/XinText.exe instead of rebuilding.

.PARAMETER SkipPandoc
  Do not bundle pandoc even if a binary is found.

.PARAMETER Installer
  Also build the NSIS installer if makensis is available on PATH.

.EXAMPLE
  pwsh ./scripts/package-windows.ps1
  pwsh ./scripts/package-windows.ps1 -Pandoc C:\tools\pandoc.exe -Installer
#>
[CmdletBinding()]
param(
  [string]$Pandoc = $(if ($env:PANDOC_BIN) { $env:PANDOC_BIN } else { 'D:\GitcodeProject\cankao\pandoc-3.11\pandoc.exe' }),
  [switch]$SkipBuild,
  [switch]$SkipPandoc,
  [switch]$Installer
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# wails3 must be reachable.
$wails = Get-Command wails3 -ErrorAction SilentlyContinue
if (-not $wails) {
  $gobin = Join-Path $env:USERPROFILE 'go\bin\wails3.exe'
  if (Test-Path $gobin) { $env:PATH = "$(Join-Path $env:USERPROFILE 'go\bin');$env:PATH" }
  else { throw 'wails3 not found on PATH. Install the Wails v3 CLI.' }
}

# Project version from build/config.yml.
$version = '0.0.0'
$config = Get-Content (Join-Path $root 'build\config.yml') -Raw
if ($config -match 'version:\s*"([^"]+)"') { $version = $Matches[1] }
$arch = $env:PROCESSOR_ARCHITECTURE.ToLower().Replace('amd64', 'amd64')
if ($arch -eq 'arm64' -or $arch -eq 'aarch64') { $arch = 'arm64' } else { $arch = 'amd64' }

if (-not $SkipBuild) {
  Write-Host '==> Production build (wails3 build)' -ForegroundColor Cyan
  wails3 build
  if ($LASTEXITCODE -ne 0) { throw 'wails3 build failed' }
}

$exe = Join-Path $root 'bin\XinText.exe'
if (-not (Test-Path $exe)) { throw "Missing $exe" }

# Stage the portable tree.
$appDir = "XinText-$version-windows-$arch"
$stage = Join-Path $root "bin\$appDir"
if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
New-Item -ItemType Directory -Path $stage | Out-Null
Copy-Item $exe (Join-Path $stage 'XinText.exe')

if (-not $SkipPandoc -and $Pandoc -and (Test-Path $Pandoc)) {
  $pdir = Join-Path $stage 'pandoc'
  New-Item -ItemType Directory -Path $pdir | Out-Null
  Copy-Item $Pandoc (Join-Path $pdir 'pandoc.exe')
  Write-Host "==> Bundled pandoc from $Pandoc" -ForegroundColor Green
} elseif (-not $SkipPandoc) {
  Write-Warning 'No pandoc.exe bundled; HTML/DOCX/TXT export needs pandoc on PATH or XinText_PANDOC set. (PDF export uses system Microsoft Edge.)'
}

# Portable zip.
$zip = Join-Path $root "bin\$appDir-portable.zip"
if (Test-Path $zip) { Remove-Item $zip -Force }
Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zip
Write-Host "==> Portable zip: $zip" -ForegroundColor Green

# Optional NSIS installer (requires makensis on PATH).
if ($Installer) {
  $makensis = Get-Command makensis -ErrorAction SilentlyContinue
  if (-not $makensis) {
    Write-Warning 'makensis not found; skipping NSIS installer. Install NSIS and retry with -Installer.'
  } else {
    Write-Host '==> Building NSIS installer' -ForegroundColor Cyan
    wails3 task windows:package
    if ($LASTEXITCODE -ne 0) { throw 'NSIS packaging failed' }
    $inst = Join-Path $root 'build\windows\nsis\XinText-installer.exe'
    if (Test-Path $inst) {
      Copy-Item $inst (Join-Path $root "bin\XinText-$version-windows-$arch-installer.exe") -Force
      Write-Host "==> Installer copied to bin\" -ForegroundColor Green
    }
  }
}

Write-Host '==> Done. Artifacts in bin/' -ForegroundColor Cyan
Get-ChildItem (Join-Path $root 'bin') -File | Where-Object { $_.Name -like "XinText-$version*" } |
  Select-Object Name, @{ N = 'MB'; E = { [math]::Round($_.Length / 1MB, 1) } } | Format-Table
