FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем файлы go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wshub ./cmd/http

# Финальный образ
FROM alpine:latest

WORKDIR /app

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates tzdata

# Копируем бинарный файл из builder
COPY --from=builder /app/wshub .

# Копируем конфигурационные файлы
COPY --from=builder /app/configs ./configs

# Запускаем приложение
CMD ["./wshub"]