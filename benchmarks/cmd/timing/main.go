package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/wispberry-tech/grove/benchmarks"
)

func main() {
	iterations := flag.Int("n", 1000, "number of iterations per engine per scenario")
	filter := flag.String("filter", "", "only run scenarios containing this substring")
	warmup := flag.Int("warmup", 10, "number of warmup renders before measuring")
	chunks := flag.Int("chunks", 10, "number of chunks to divide iterations into (for stddev)")
	flag.Parse()

	scenarios := benchmarks.AllTimingScenarios()

	fmt.Println("══════════════════════════════════════════════════════════")
	fmt.Printf("  Grove Timing Benchmark — %d iterations (warmup: %d, chunks: %d)\n", *iterations, *warmup, *chunks)
	fmt.Println("══════════════════════════════════════════════════════════")

	for si, sc := range scenarios {
		if *filter != "" && !strings.Contains(sc.Name, *filter) {
			continue
		}

		fmt.Println()
		fmt.Printf("  %s\n", sc.Name)
		fmt.Printf("  %-18s %12s %12s %12s %12s %12s\n", "Engine", "Avg/render", "ops/sec", "Min", "Max", "±StdDev")
		fmt.Printf("  %-18s %12s %12s %12s %12s %12s\n", "──────────────────", "────────────", "────────────", "────────────", "────────────", "────────────")

		data := sc.Data()
		// Fresh engines per scenario to avoid template name/cache collisions (Jet caches by name).
		engines := benchmarks.AllEngines()
		tplName := fmt.Sprintf("timing_%d", si)

		for _, eng := range engines {
			src := sc.Templates[eng.Name()]
			d := benchmarks.EngineData(eng, data)

			// Parse + warmup renders
			if err := eng.Parse(tplName, src); err != nil {
				fmt.Fprintf(os.Stderr, "  %-18s PARSE ERROR: %v\n", eng.Name(), err)
				continue
			}
			for i := 0; i < *warmup; i++ {
				if _, err := eng.Render(tplName, d); err != nil {
					fmt.Fprintf(os.Stderr, "  %-18s WARMUP ERROR: %v\n", eng.Name(), err)
					break
				}
			}

			// Timed loop with chunk-based stddev calculation
			chunkSize := *iterations / *chunks
			if chunkSize < 1 {
				chunkSize = 1
			}
			chunkTimes := make([]time.Duration, 0, *chunks)

			for chunk := 0; chunk < *chunks; chunk++ {
				start := time.Now()
				for i := 0; i < chunkSize; i++ {
					if _, err := eng.Render(tplName, d); err != nil {
						fmt.Fprintf(os.Stderr, "  %-18s ERROR at iteration %d: %v\n", eng.Name(), chunk*chunkSize+i, err)
						break
					}
				}
				chunkTimes = append(chunkTimes, time.Since(start))
			}

			// Calculate statistics from chunk timings
			var totalTime time.Duration
			minTime := time.Duration(math.MaxInt64)
			maxTime := time.Duration(0)
			for _, ct := range chunkTimes {
				totalTime += ct
				if ct < minTime {
					minTime = ct
				}
				if ct > maxTime {
					maxTime = ct
				}
			}

			// Scale chunk times to per-iteration (chunk size / chunk time = ops per chunk)
			avgTime := totalTime / time.Duration(*iterations)

			// Compute stddev of chunk times
			mean := float64(totalTime) / float64(len(chunkTimes))
			var variance float64
			for _, ct := range chunkTimes {
				diff := float64(ct) - mean
				variance += diff * diff
			}
			variance /= float64(len(chunkTimes))
			stddev := time.Duration(math.Sqrt(variance))

			opsPerSec := float64(*iterations) / totalTime.Seconds()

			fmt.Printf("  %-18s %12s %12s %12s %12s %12s\n",
				eng.Name(),
				formatDuration(avgTime),
				formatOps(opsPerSec),
				formatDuration(minTime/time.Duration(chunkSize)),
				formatDuration(maxTime/time.Duration(chunkSize)),
				formatDuration(stddev/time.Duration(chunkSize)))
		}
	}

	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════════════")
}

func formatOps(ops float64) string {
	switch {
	case ops >= 1_000_000:
		return fmt.Sprintf("%.1fM", ops/1_000_000)
	case ops >= 1_000:
		return fmt.Sprintf("%.1fK", ops/1_000)
	default:
		return fmt.Sprintf("%.0f", ops)
	}
}

func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	case d >= time.Microsecond:
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1e3)
	default:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}
