# Estágio 1: Builder
FROM golang:1.24-alpine AS builder

# Instalar dependências de sistema (certificados SSL e timezone)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Inicialmente copiando go.mod (não temos go.sum ainda, pois não há deps externas explicitas)
COPY go.mod ./
RUN go mod download

# Copiando código fonte
COPY . .

# Build estático (CGO_ENABLED=0 garante um binário independente de C libraries)
# Usando flags de linker (-s -w) para diminuir binário
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o chaos-target main.go

# Estágio 2: Final (Imagem Mínima)
FROM scratch

# Copiar certificados ca-certificates da imagem builder (Para o Service Chaining HTTPs outbound)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copiar binário compilado
COPY --from=builder /app/chaos-target /chaos-target

# Variáveis default de ambiente
ENV PORT=8080
ENV MIN_DELAY_MS=10
ENV MAX_DELAY_MS=100
ENV BURN_CPU=false
ENV CPU_COMPLEXITY=50000
ENV EXTERNAL_SERVICES=""
ENV MAX_CALL_DEPTH=5
ENV REQUEST_TIMEOUT=5

EXPOSE 8080

# Executar a app
ENTRYPOINT ["/chaos-target"]
