#!/bin/bash
# 🧬 [CÓRTEX]: Supervisor Soberano de Procesos

# 1. Iniciar Ollama en segundo plano
ollama serve &

# 2. Esperar conexión real a la infraestructura de IA
echo "⏳ Esperando infraestructura de IA..."
until curl -s http://localhost:11434/api/tags > /dev/null; do
  sleep 2
done

# 3. Pull del modelo (Aseguramos disponibilidad)
ollama pull llama3

# 4. Bucle de Resiliencia (El sistema nunca debe morir)
while true; do
    echo "🚀 [CÓRTEX]: Iniciando motor principal..."
    
    # Reporte de Salud al Buzón (Antes de arrancar)
    curl -X POST https://geochat-buzon.onrender.com/api/enviar \
         -H "Content-Type: application/json" \
         -d '{"mensaje": "Córtex reiniciado - Estado: OK", "tipo": "KIMI"}'

    # Limpiamos posibles residuos de compilaciones fallidas anteriores
    # para garantizar que cada arranque sea una hoja en blanco
    rm -f ./sandbox/test_run
    
    # Ejecutar el binario (compilado por el builder en el Dockerfile)
    /usr/local/bin/geochat-core
    
    # Si llegamos aquí, el binario se detuvo por error o evolución
    echo "⚠️ [CÓRTEX]: Proceso terminado. Reiniciando en 5 segundos..."
    sleep 5
done