#!/bin/bash
# scripts/test_evolucion.sh
echo "🔍 [CÓRTEX]: Validando evolución..."

# 1. Copiar al directorio raíz para compilar correctamente con el go.mod
cp ./sandbox/tmp_evolucion.go main.go

# 2. Intentar compilar el proyecto completo
go build -o ./sandbox/test_run . > ./sandbox/build.log 2>&1

if [ $? -eq 0 ]; then
    echo "✅ [CÓRTEX]: Evolución estable."
    rm ./sandbox/test_run
    exit 0
else
    echo "❌ [CÓRTEX]: Evolución inestable. Revisa build.log"
    exit 1
fi