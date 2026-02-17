.PHONY: test-all test test-bench test-fuzz test-race test-cover

test-all: test test-race test-bench test-fuzz

test:
	go test ./...

test-race:
	go test ./... -race -count=1

test-bench:
	go test ./... -run=^$$ -bench=. -benchmem -cpu=1,2,4,8 -count=1

test-fuzz:
	go test ./internal/service/courier_pool -run=^$$ -fuzz=FuzzCourierPoolAcquireRelease -fuzztime=10s -count=1


test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
