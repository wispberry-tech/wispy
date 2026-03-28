#!/bin/bash
set -e

echo "Running Wisp-only benchmarks..."
go test -bench='_Wisp$' ./pkg/engine/... -benchtime=100ms -count=1 2>&1 | grep '^Benchmark'
