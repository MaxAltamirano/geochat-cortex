#!/bin/bash
echo "🧬 [CÓRTEX]: Iniciando ritual de despertar..."

# 1. Limpiar registros previos
> cortex.log

# 2. Asegurar que las dependencias están sincronizadas
go mod tidy
go mod vendor

# 3. Compilar el Córtex
echo "⚙️ [CÓRTEX]: Compilando núcleo..."
go build -o main main.go

# 4. Lanzar el ejecutable en segundo plano
./main &

echo "🚀 [CÓRTEX]: Sistema online. Puerto 10000 activo."
