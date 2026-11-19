# Instala el CLI de Wails fijado para este proyecto en .tools\ (sin tocar el CLI global).
# Motivo: Wails CLI v2.10.x se compiló con golang.org/x/tools v0.30, que no lee paquetes de Go 1.27.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$env:GOBIN = Join-Path $root ".tools"
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
Write-Host "Wails instalado en $env:GOBIN\wails.exe"
