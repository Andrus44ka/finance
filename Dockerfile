# ── Этап 1: сборка ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем зависимости и загружаем модули (кэшируется отдельным слоем)
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Копируем весь исходный код
COPY . .

# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go

# ── Этап 2: минимальный образ для запуска ────────────────────────────────────
FROM alpine:3.21

WORKDIR /app

# Копируем бинарник из этапа сборки
COPY --from=builder /app/server .

# Копируем миграции
COPY --from=builder /app/migrations ./migrations

COPY --from=builder /go/bin/goose /usr/local/bin/goose


# Копируем веб-файлы (HTML, CSS, JS)
COPY --from=builder /app/web ./web

EXPOSE 80

CMD ["sh", "start.sh"]