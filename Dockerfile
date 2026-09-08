FROM golang:1.27.1-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/vk_backend ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/vk_worker ./cmd/worker

FROM alpine:latest AS api
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /out/vk_backend .
EXPOSE 8080
CMD ["./vk_backend"]

FROM alpine:latest AS worker
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /out/vk_worker .
CMD ["./vk_worker"]
