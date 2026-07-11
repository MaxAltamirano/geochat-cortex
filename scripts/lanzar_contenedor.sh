#!/bin/bash
# Script para lanzar el contenedor con persistencia total y limpieza previa

# 1. Si el contenedor ya existe, lo detenemos y eliminamos para asegurar una instancia limpia
if [ "$(docker ps -aq -f name=geochat-cortex-container)" ]; then
    echo "⚠️ [CÓRTEX]: Contenedor existente detectado. Limpiando..."
    docker rm -f geochat-cortex-container
fi

echo "🚀 [CÓRTEX]: Lanzando nueva instancia soberana..."

# 2. Lanzar el contenedor
docker run -d \
  --name geochat-cortex-container \
  -v ~/Documentos/Geochat-Lab/geochat-core:/app \
  -p 10000:10000 \
  geochat-cortex-image

echo "✅ [CÓRTEX]: Sistema online en puerto 10000."