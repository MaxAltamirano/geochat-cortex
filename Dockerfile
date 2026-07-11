# 1. Etapa de Construcción
FROM golang:latest AS builder
WORKDIR /app
COPY . .
# Compilamos el núcleo soberano
RUN CGO_ENABLED=0 GOOS=linux go build -o geochat-core .

# 2. Imagen final
FROM ollama/ollama:latest

# Instalar herramientas necesarias para la resiliencia (curl para healthchecks)
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

# Definir el directorio de trabajo donde montaremos el volumen local
WORKDIR /app

# Copiar el binario desde el builder
COPY --from=builder /app/geochat-core /usr/local/bin/geochat-core

# Exponer el puerto del Córtex
EXPOSE 10000

# Preparar el script de arranque
COPY entrypoint.sh /entrypoint.sh

# Configurar permisos soberanos
# Ajustamos permisos sobre /app para que el volumen montado sea escribible
RUN chown -R 1000:1000 /app && chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]