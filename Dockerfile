# Base Image
FROM golang:1.19-alpine as builder
ARG SERVICE_NAME
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 go build -o ${SERVICE_NAME} ./cmd/api
RUN chmod +x /app/${SERVICE_NAME}

# Tiny Image
FROM alpine:latest
ARG SERVICE_NAME
ENV APP_PATH /app/${SERVICE_NAME}
WORKDIR /app
COPY --from=builder /app/${SERVICE_NAME} /app
CMD ${APP_PATH}