#!/bin/bash
# Iniciar Ollama en segundo plano
ollama serve &

# Esperar a que Ollama responda (mucho mejor que sleep)
echo "Esperando a que Ollama inicie..."
until curl -s http://localhost:11434/api/tags > /dev/null; do
  sleep 2
done

ollama pull llama3
exec /usr/local/bin/geochat-core
