# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# git is needed for some Go module fetches
RUN apk add --no-cache git

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Build the server (pure Go, CGO disabled -> static binary, alpine-friendly)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20
WORKDIR /app

# ca-certificates: outbound TLS (Gemini, Stripe, Google, SMTP)
# tzdata: correct timezone handling
RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /data

COPY --from=build /out/server /app/server

# WhatsApp SQLite session store lives on a persisted volume
ENV WHATSAPP_DB_PATH="file:/data/whatsapp_sessions.db?_foreign_keys=on"

EXPOSE 8081
CMD ["/app/server"]
