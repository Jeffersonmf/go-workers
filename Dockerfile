# syntax=docker/dockerfile:1

# --- builder ---------------------------------------------------------------
FROM golang:1.27-bookworm AS builder
ARG TARGETARCH
WORKDIR /app

# Camada de dependências isolada: só rebuilda se os manifests mudarem.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0: nenhuma dependência deste módulo usa cgo (viper, zap,
# fsnotify, google/uuid são Go puro), então o binário já sai estático
# sem precisar de musl-tools ou de um target Rust-style separado — só
# desligar cgo e fixar GOARCH a partir do TARGETARCH do BuildKit.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o /app/go-workers .

# --- runtime -----------------------------------------------------------
# Binário estático: scratch é suficiente, sem shell, sem libc dinâmica.
FROM scratch
COPY --from=builder /app/go-workers /go-workers
ENTRYPOINT ["/go-workers"]
