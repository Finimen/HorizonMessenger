FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./app

FROM alpine:latest
RUN apk --no-cache add ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/app/static ./static
COPY --from=builder /app/app/config ./config
COPY --from=builder /app/docs ./docs

RUN chown -R appuser:appgroup /root/

USER appuser

EXPOSE 8080

RUN chmod +x main

# Исправляем проблему с портом
ENV SERVER_PORT=8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1
   
CMD ["./main"]