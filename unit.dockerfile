# Tiny Image
FROM golang:1.19-alpine
WORKDIR /app
COPY . /app
ENV GO111MODULE=on
CMD go run ./cmd/api/main.go