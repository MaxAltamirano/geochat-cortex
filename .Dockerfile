# 1. Etapa de Construcción (usamos debian para compatibilidad total con ollama)
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
# Compilamos como estático para evitar dependencias de librerías
RUN CGO_ENABLED=0 GOOS=linux go build -o geochat-core .

# 2. Imagen final (Base sólida)
FROM ollama/ollama:latest

# Instalar herramientas necesarias
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

# Copiar el binario desde el builder
COPY --from=builder /app/geochat-core /usr/local/bin/geochat-core

# Exponemos el puerto de tu API (Asegúrate que coincida con tu main.go)
EXPOSE 10000

# Script de arranque
# Usamos un script porque CMD solo permite un proceso principal
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]