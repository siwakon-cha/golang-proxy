FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rpc-proxy .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /root/

COPY --from=builder /app/rpc-proxy .

# Set default environment variables
ENV SERVER_PORT=8888
ENV DB_HOST=localhost
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=password
ENV DB_NAME=rpc_proxy
ENV DB_SSLMODE=disable
ENV HEALTH_CHECK_INTERVAL=30s
ENV HEALTH_CHECK_TIMEOUT=5s
ENV HEALTH_CHECK_RETRIES=3
ENV PROXY_TIMEOUT=10s
ENV PROXY_MAX_CONNECTIONS=1000
ENV APP_ENV=production
ENV LOG_LEVEL=info

EXPOSE ${SERVER_PORT}

CMD ["./rpc-proxy"]