# Build stage
FROM golang:1.26.9-alpine3.24 AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/bottagond ./cmd/bottagond
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/doctor ./cmd/doctor
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/refbot ./cmd/refbot

# Final release stage - hardened alpine
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 bottagon \
    && adduser -S -u 10001 -G bottagon -H -s /sbin/nologin bottagon

WORKDIR /app

COPY --from=builder --chown=bottagon:bottagon /bin/bottagond /app/bottagond
COPY --from=builder --chown=bottagon:bottagon /bin/migrate /app/migrate
COPY --from=builder --chown=bottagon:bottagon /bin/doctor /app/doctor
COPY --from=builder --chown=bottagon:bottagon /bin/refbot /app/refbot
COPY --chown=bottagon:bottagon migrations /app/migrations

USER bottagon:bottagon

EXPOSE 8080

ENTRYPOINT ["/app/bottagond"]
