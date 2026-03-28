#!/bin/bash
set -e

echo "Running cross-engine benchmark comparisons..."
go test -bench='^BenchmarkRenderString(Simple|WithConditionals|WithLoop|WithNestedAccess|WithFilters|Complex)(Wisp|TextTemplate|HtmlTemplate|Pongo2)$|^Benchmark(AutoEscape|Caching)(Wisp|HtmlTemplate|Pongo2)$' ./pkg/engine/... -benchtime=100ms -run='^$' -count=1 2>&1 | grep '^Benchmark'
