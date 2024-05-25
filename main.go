package main

import (
	"CPUProfiler/collector"
	"CPUProfiler/testutil"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type MyCollector struct {
	mu      sync.Mutex
	records map[string]uint32
}

func NewMyCollector() *MyCollector {
	return &MyCollector{
		records: make(map[string]uint32),
	}
}

func (c *MyCollector) Collect(stats []collector.CPUTimeRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, stat := range stats {
		c.records[stat.Label] += stat.CPUTimeMs
	}
	c.printStats()
}

func (c *MyCollector) printStats() {
	fmt.Println("Current CPU Usage by Rule:")
	totalCPUTime := uint32(0)
	for _, cpuTime := range c.records {
		totalCPUTime += cpuTime
	}

	if totalCPUTime == 0 {
		fmt.Println("No CPU time recorded")
		return
	}

	keys := make([]string, 0, len(c.records))
	for k := range c.records {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return c.records[keys[i]] > c.records[keys[j]]
	})

	for _, label := range keys {
		cpuTime := c.records[label]
		fmt.Printf("Rule: %s, CPU Time: %d ms, Percentage: %.2f%%\n", label, cpuTime, float64(cpuTime)/float64(totalCPUTime)*100)
	}
	fmt.Println()
}

func main() {
	myCollector := NewMyCollector()
	cc := collector.NewCPUCollector(myCollector)

	cc.Start()
	defer cc.Stop()

	done := make(chan struct{})

	var wg sync.WaitGroup
	startSignal := make(chan struct{})

	singleLabelRules := []string{"rule1", "rule2", "rule3"}
	for _, rule := range singleLabelRules {
		wg.Add(1)
		go func(rule string) {
			defer wg.Done()
			<-startSignal
			ctx := collector.CtxWithLabel(context.Background(), rule)
			testutil.MockSingleCPULoad(ctx, rule)
		}(rule)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startSignal
		labels := []string{"rule1", "rule2", "rule3"}
		ctx := collector.CtxWithLabels(context.Background(), labels)
		testutil.MockMultiCPULoad(ctx, labels...)
	}()

	close(startSignal)

	time.Sleep(10 * time.Second)
	close(done)

	wg.Wait()

	time.Sleep(1 * time.Second)
	fmt.Println("Main exiting")
}
