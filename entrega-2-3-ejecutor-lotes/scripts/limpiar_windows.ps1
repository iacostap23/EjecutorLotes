$ErrorActionPreference = "SilentlyContinue"

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "========================================"
Write-Host " LIMPIEZA WINDOWS - EJECUTOR DE LOTES"
Write-Host "========================================"
Write-Host ""

Write-Host "==> Deteniendo procesos del proyecto si estan activos..."

Stop-Process -Name gesfich,gesprog,ejecutor,ctrllt,cliente,procesar,dormir -Force -ErrorAction SilentlyContinue

Start-Sleep -Milliseconds 500

Write-Host "==> Eliminando carpetas generadas..."

Remove-Item -Recurse -Force ".\aralmac" -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force ".\bin" -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force ".\logs" -ErrorAction SilentlyContinue

Write-Host "==> Eliminando archivos generados..."

Remove-Item ".\entrada.txt" -Force -ErrorAction SilentlyContinue


Remove-Item ".\programas\procesar.exe" -Force -ErrorAction SilentlyContinue
Remove-Item ".\programas\dormir.exe" -Force -ErrorAction SilentlyContinue


Remove-Item ".\programas\procesar" -Force -ErrorAction SilentlyContinue
Remove-Item ".\programas\dormir" -Force -ErrorAction SilentlyContinue


Remove-Item ".\programas\dormir.go" -Force -ErrorAction SilentlyContinue

Write-Host "==> Eliminando carpeta de pruebas generada..."

Remove-Item -Recurse -Force ".\pruebas\dormir" -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Limpieza finalizada correctamente." -ForegroundColor Green
Write-Host ""
Write-Host "Se conservaron los archivos fuente, scripts y documentos."