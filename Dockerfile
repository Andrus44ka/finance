# ── Этап 1: сборка ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder
 
WORKDIR /app
 
# Копируем зависимости и загружаем модули (кэшируется отдельным слоем)
COPY go.mod go.sum ./
RUN go mod download
 
# Копируем весь исходный код
COPY . .
 
# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go
 
# ── Этап 2: минимальный образ для запуска ────────────────────────────────────
FROM alpine:3.19
 
WORKDIR /app
 
# Копируем бинарник из этапа сборки
COPY --from=builder /app/server .
 
# Копируем веб-файлы (HTML, CSS, JS)
COPY --from=builder /app/web ./web
 
EXPOSE 8080
 
CMD ["./server"]