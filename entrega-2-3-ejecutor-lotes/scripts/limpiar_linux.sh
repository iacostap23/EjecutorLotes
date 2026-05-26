#!/usr/bin/env bash

# scripts/limpiar_linux.sh
# Limpia archivos generados del proyecto en Linux / WSL.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT" || exit 1

echo "========================================"
echo " LIMPIEZA LINUX / WSL - EJECUTOR DE LOTES"
echo "========================================"
echo ""

echo "==> Deteniendo procesos del proyecto si estan activos..."

pkill -x gesfich 2>/dev/null || true
pkill -x gesprog 2>/dev/null || true
pkill -x ejecutor 2>/dev/null || true
pkill -x ctrllt 2>/dev/null || true
pkill -x cliente 2>/dev/null || true
pkill -x procesar 2>/dev/null || true
pkill -x dormir 2>/dev/null || true

sleep 0.5

echo "==> Eliminando tuberias temporales..."

rm -f /tmp/lotes-* 2>/dev/null || true

echo "==> Eliminando carpetas generadas..."

if [ -d "./aralmac" ]; then
    chmod -R u+rwX ./aralmac 2>/dev/null || true
    rm -rf ./aralmac 2>/tmp/error_limpiar_aralmac.txt || true

    if [ -d "./aralmac" ]; then
        echo ""
        echo "No se pudo borrar ./aralmac completamente."
        echo "Detalle:"
        cat /tmp/error_limpiar_aralmac.txt 2>/dev/null || true
        echo ""
        echo "Prueba manualmente con:"
        echo "  sudo rm -rf ./aralmac"
        echo ""
    fi
fi

rm -rf ./bin 2>/dev/null || true
rm -rf ./logs 2>/dev/null || true

echo "==> Eliminando archivos generados..."

rm -f ./entrada.txt 2>/dev/null || true

rm -f ./programas/procesar 2>/dev/null || true
rm -f ./programas/dormir 2>/dev/null || true
rm -f ./programas/procesar.exe 2>/dev/null || true
rm -f ./programas/dormir.exe 2>/dev/null || true
rm -f ./programas/dormir.go 2>/dev/null || true

echo "==> Eliminando carpeta de pruebas generada..."

rm -rf ./pruebas/dormir 2>/dev/null || true

echo ""
echo "Limpieza finalizada correctamente."
echo ""
echo "Se conservaron los archivos fuente, scripts y documentos."