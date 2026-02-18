.PHONY: test-all test test-bench test-fuzz \
		test-race test-cover vet lint fmt run 

# tests 
test-all: test test-race test-bench test-fuzz

test:
	go test ./...

test-race:
	go test ./... -race -count=1

test-bench:
	go test ./... -run=^$$ -bench=. -benchmem -cpu=1,2,4,8 -count=1

test-fuzz:
	go test ./internal/service/pool -run=^$$ -fuzz=FuzzPoolAcquireRelease -fuzztime=10s -count=1


test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

# https://go.dev/src/cmd/vet/doc.go
vet:
	go vet ./...

# https://golangci-lint.run/docs/
lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

# run
run:
	go run ./cmd/server