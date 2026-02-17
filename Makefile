.PHONY: all test test-bench test-fuzz test-clean test-race

test-all: test test-bench test-fuzz

test:
	mkdir -p artifacts
	@ts=$$(date '+%Y%m%d-%H%M%S'); \
	out="artifacts/test-$$ts.txt"; \
	echo "Run at: $$(date '+%Y-%m-%d %H:%M:%S %Z')" | tee "$$out"; \
	go test ./... | tee -a "$$out"

test-bench:
	mkdir -p artifacts
	@ts=$$(date '+%Y%m%d-%H%M%S'); \
	out="artifacts/bench-$$ts.txt"; \
	echo "Run at: $$(date '+%Y-%m-%d %H:%M:%S %Z')" | tee "$$out"; \
	go test ./internal/service/courier_pool -bench=. -benchmem -cpu=1,2,4,8 | tee -a "$$out"

test-fuzz:
	mkdir -p artifacts
	@ts=$$(date '+%Y%m%d-%H%M%S'); \
	out="artifacts/fuzz-$$ts.txt"; \
	echo "Run at: $$(date '+%Y-%m-%d %H:%M:%S %Z')" | tee "$$out"; \
	go test ./internal/service/courier_pool -run=^$$ -fuzz=FuzzCourierPoolAcquireRelease -fuzztime=10s | tee -a "$$out"

test-clean:
	rm -rf artifacts/*

test-race:
	go test ./... -race
