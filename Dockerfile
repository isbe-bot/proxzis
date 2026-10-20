# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/bottagond ./cmd/bottagond
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/migrate ./cmd/migrate

# Final stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/bottagond /app/bottagond
COPY --from=builder /bin/migrate /app/migrate
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/bottagond"]
