.PHONY: all test bench fuzz

all: test fuzz bench

test:
	mkdir -p artifacts
	@ts=$$(date '+%Y%m%d-%H%M%S'); \
	out="artifacts/test-$$ts.txt"; \
	echo "Run at: $$(date '+%Y-%m-%d %H:%M:%S %Z')" | tee "$$out"; \
	go test ./... | tee -a "$$out"

bench:
	mkdir -p artifacts
	@ts=$$(date '+%Y%m%d-%H%M%S'); \
	out="artifacts/bench-$$ts.txt"; \
	echo "Run at: $$(date '+%Y-%m-%d %H:%M:%S %Z')" | tee "$$out"; \
	go test ./internal/service/courier_pool -bench=. -benchmem -cpu=1,2,4,8 | tee -a "$$out"

fuzz:
	mkdir -p artifacts
	@ts=$$(date '+%Y%m%d-%H%M%S'); \
	out="artifacts/fuzz-$$ts.txt"; \
	echo "Run at: $$(date '+%Y-%m-%d %H:%M:%S %Z')" | tee "$$out"; \
	go test ./internal/service/courier_pool -run=^$$ -fuzz=FuzzCourierPoolAcquireRelease -fuzztime=10s | tee -a "$$out"