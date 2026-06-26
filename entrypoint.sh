#!/bin/bash
# 🧬 [CÓRTEX]: Supervisor Soberano de Procesos

# 1. Iniciar Ollama
ollama serve &

# 2. Esperar conexión real
echo "⏳ Esperando infraestructura de IA..."
until curl -s http://localhost:11434/api/tags > /dev/null; do
  sleep 2
done

# 3. Pull del modelo
ollama pull llama3

# 4. Bucle de Resiliencia (El sistema nunca debe morir)
while true; do
    echo "🚀 [CÓRTEX]: Iniciando motor principal..."
    
    # Reporte de Salud al Buzón (Antes de arrancar)
    curl -X POST https://geochat-buzon.onrender.com/api/enviar \
         -H "Content-Type: application/json" \
         -d '{"mensaje": "Córtex reiniciado - Estado: OK", "tipo": "KIMI"}'

    # Ejecutar el binario
    /usr/local/bin/geochat-core
    
    echo "⚠️ [CÓRTEX]: El proceso murió. Reiniciando en 5 segundos..."
    sleep 5
done
