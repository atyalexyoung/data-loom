# Stage 1: Build the server
FROM golang:1.21 AS builder
WORKDIR /app
COPY ./server ./server
COPY go.mod go.sum ./
RUN go mod download
RUN go build -o myserver ./server/service/main.go

# Stage 2: Runtime
FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/myserver .
EXPOSE 50051
CMD ["./myserver"]