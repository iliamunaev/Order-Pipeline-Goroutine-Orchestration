.PHONY: ci test test-race test-bench test-fuzz test-cover vet lint fmt run

ci: fmt vet lint test-race

test:
	go test ./...

test-race:
	go test ./... -race -count=1

test-bench:
	go test ./... -run=^$$ -bench=. -benchmem -cpu=1,2,4,8 -count=1

test-fuzz:
	go test ./internal/transport/http -run=^$$ -fuzz=FuzzHandleOrder -fuzztime=10s -count=1

test-cover:
	go test ./internal/order/... ./internal/service/... ./internal/transport/... -coverprofile=coverage.out
	go tool cover -func=coverage.out

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

run:
	go run ./cmd/server