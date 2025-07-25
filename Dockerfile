FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o wpp-wave-bot ./cmd

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/wpp-wave-bot /usr/local/bin/wpp-wave-bot
COPY config.yaml /app/config.yaml
ENTRYPOINT ["/usr/local/bin/wpp-wave-bot"]
CMD ["run"]
