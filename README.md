# Order Pipeline

Minimal Go HTTP service that simulates an order workflow with payment, vendor
notification, and courier assignment steps. Each step can be delayed or forced
to fail for testing.

## Requirements

- Go 1.22+ (module uses `go` toolchain)

## Run

```sh
go run ./cmd/server
```

Server listens on `:8080`.

## Endpoints

- `GET /health` -> `200 ok`
- `POST /order` -> processes an order

### Order request body

```json
{
  "order_id": "o1",
  "amount": 1200,
  "fail_step": "payment",
  "delay_ms": {
    "payment": 150,
    "vendor": 200,
    "courier": 100
  }
}
```

- `fail_step` can be `payment`, `vendor`, or `courier`
- `delay_ms` overrides per-step delays in milliseconds

### Examples

```sh
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o1","amount":10}'
```

```sh
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o2","amount":10,"fail_step":"payment","delay_ms":{"vendor":800,"courier":800}}'
```

```sh
curl -i -X POST http://localhost:8080/order \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"o3","amount":10,"delay_ms":{"payment":3000,"vendor":3000,"courier":3000}}'
```

## Logging

Requests are logged to `server.log` in the repo root and to stdout. Each entry
includes a request id, method, path, status, response bytes, and duration.

