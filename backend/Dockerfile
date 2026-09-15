# ─── Build stage ───
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Static binary, tanpa cgo (lib/pq pure Go, aman)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /server ./cmd

# ─── Run stage ───
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /server /app/server
EXPOSE 8080
CMD ["/app/server"]
