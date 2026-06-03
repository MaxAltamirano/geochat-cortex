#!/bin/bash
# Iniciar Ollama en segundo plano
ollama serve &
# Esperar a que Ollama levante (puedes ajustar el tiempo)
sleep 10
# Descargar el modelo
ollama pull llama3
# Ejecutar tu aplicación Go
exec /usr/local/bin/geochat-core
