# ── build stage ───────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY . .

# download deps
RUN go mod download

# compile the *example* program (static binary, CGO disabled)
RUN CGO_ENABLED=0 go build -o /app/telegram-example ./example

# ── tiny runtime stage ────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/telegram-example /app/telegram-example

# create log directory
RUN mkdir -p /app/logs

# expose any needed ports (none for the sender)
# EXPOSE 8080

# run the example bot
CMD ["/app/telegram-example"]