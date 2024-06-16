package testutil

import (
	"context"
	"math"
	"runtime/pprof"
)

// MockSingleCPULoad A goroutine with only one label
func MockSingleCPULoad(ctx context.Context, label string) {
	go mockCPULoadByGoroutineWithLabel(ctx, "label", label)
}

// MockMultiCPULoad A goroutine with multiple labels
func MockMultiCPULoad(ctx context.Context, labels ...string) {
	lvs := []string{}
	for _, label := range labels {
		lvs = append(lvs, "label", label)
	}
	go mockCPULoadByGoroutineWithLabel(ctx, lvs...)
}

func mockCPULoadByGoroutineWithLabel(ctx context.Context, labels ...string) {
	ctx = pprof.WithLabels(ctx, pprof.Labels(labels...))
	pprof.SetGoroutineLabels(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			cpuIntensiveTask()
		}
	}
}

func cpuIntensiveTask() {
	primeCount := 0
	for i := 2; i < 1000000; i++ {
		isPrime := true
		for j := 2; j <= int(math.Sqrt(float64(i))); j++ {
			if i%j == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			primeCount++
		}
	}
}
