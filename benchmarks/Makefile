.PHONY: bench compare bench-grove compare-runs timing clean

BENCH_FLAGS ?= -benchmem -count=6 -timeout=10m
RESULTS_DIR := results
TIMESTAMP := $(shell date +%Y%m%d-%H%M%S)

# Run all benchmarks and save raw output to results/ with timestamp.
bench:
	@mkdir -p $(RESULTS_DIR)
	go test -bench=. -run='^$$' $(BENCH_FLAGS) | tee $(RESULTS_DIR)/bench-$(TIMESTAMP).txt
	@ln -sf bench-$(TIMESTAMP).txt $(RESULTS_DIR)/latest.txt
	@echo ""
	@echo "Results saved to $(RESULTS_DIR)/bench-$(TIMESTAMP).txt"

# Format results with benchstat (install: go install golang.org/x/perf/cmd/benchstat@latest).
compare:
	@command -v benchstat >/dev/null 2>&1 || { echo "Install benchstat: go install golang.org/x/perf/cmd/benchstat@latest"; exit 1; }
	@test -f $(RESULTS_DIR)/latest.txt || { echo "No results found. Run 'make bench' first."; exit 1; }
	benchstat $(RESULTS_DIR)/latest.txt

# Compare two benchmark runs.
compare-runs:
	@command -v benchstat >/dev/null 2>&1 || { echo "Install benchstat: go install golang.org/x/perf/cmd/benchstat@latest"; exit 1; }
	@test -n "$(OLD)" -a -n "$(NEW)" || { echo "Usage: make compare-runs OLD=results/bench-old.txt NEW=results/bench-new.txt"; exit 1; }
	benchstat $(OLD) $(NEW)

# Run only Grove benchmarks for quick regression checks (uses -bench '/Grove' for efficiency).
bench-grove:
	@mkdir -p $(RESULTS_DIR)
	go test -bench='/Grove' -run='^$$' $(BENCH_FLAGS) -benchtime=3s | tee $(RESULTS_DIR)/grove-$(TIMESTAMP).txt
	@ln -sf grove-$(TIMESTAMP).txt $(RESULTS_DIR)/grove-latest.txt
	@echo ""
	@echo "Results saved to $(RESULTS_DIR)/grove-$(TIMESTAMP).txt"

# Run wall-clock timing benchmarks across all engines (large templates).
# Use -f to filter by scenario name: make timing FILTER="Large Loop"
timing:
	./run-timing.sh

clean:
	rm -rf $(RESULTS_DIR)
