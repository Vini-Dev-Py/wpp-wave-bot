# wpp-wave-bot

`wpp-wave-bot` is a microservice responsible for sending and receiving WhatsApp
messages for multiple companies. It communicates using RabbitMQ and stores
messages and contacts in PostgreSQL. WhatsApp connectivity is implemented with
[whatsmeow](https://github.com/tulir/whatsmeow).

## Running locally

```bash
# build containers and start services
docker-compose up --build
```

This will start PostgreSQL, RabbitMQ and the bot itself. The bot connects to the
queues `wpp:send` and `wpp:received`. Session events are published to the
`wpp:sessions` queue.

## Migrations and seeds

Database migrations are executed automatically on startup. You can run the seed
command manually:

```bash
docker-compose run --rm bot seed
```

### Using the Makefile

For convenience a `Makefile` includes common commands:

```bash
make run       # start services with docker-compose
make migrate   # run migrations once
make seed      # execute database seeds
```

## Project structure

- `cmd/` – application entry point
- `internal/db/` – database connection, migrations and seeds
- `internal/rabbitmq/` – RabbitMQ wrapper
- `internal/whatsapp/` – WhatsApp session manager and client logic
- `scripts/` – helper scripts

## Message flow

1. Messages to send are published to the `wpp:send` queue.
2. The bot will dispatch them through WhatsApp and store them in the database.
3. Received messages will be pushed to the `wpp:received` queue for the
   orchestrator.


You can start a session by hitting the `/sessions/{id}/connect` endpoint which
returns the QR code as base64. Once authenticated the session will be restored
on the next start.

## Admin API

An HTTP server is exposed on port `8080` to help manage sessions and manual
message sending.

- `GET /sessions` – list active company IDs
- `POST /sessions/{id}/connect` – create a session and get the QR code (base64)
- `POST /sessions/{id}/logout` – force logout a company session
- `POST /messages` – send a message body directly using JSON
- `GET /health` – simple health check

Example payload for `/messages`:

```json
{
  "company_id": "empresa-123",
  "type": "text",
  "to": "5511999999999@c.us",
  "message": "Olá"
}
```
