# Developer Notes

## Tests

Run all tests:

```sh
go test ./...
```

Skip stress tests:

```sh
go test ./... -short
```

Run only the stress test:

```sh
go test ./internal/handler -run Stress
```

## Curl smoke tests

The `curl_tests.sh` script can start the server, wait for readiness, and run
basic request checks:

```sh
bash curl_tests.sh
```

It also accepts a custom base URL:

```sh
bash curl_tests.sh http://localhost:8080
```

## Dependency notes

The handler uses `golang.org/x/sync/errgroup` for concurrent step execution.

