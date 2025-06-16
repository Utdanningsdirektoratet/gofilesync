FROM golang:1.24-alpine3.22 AS builder

WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o gofilesync ./cmd/gofilesync

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/gofilesync /usr/local/bin/
ENTRYPOINT ["sh"]
