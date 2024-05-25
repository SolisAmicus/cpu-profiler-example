package collector

import (
	"context"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/google/pprof/profile"
)

const (
	labelKey = "label"
)

type CPUTimeRecord struct {
	Label     string
	CPUTimeMs uint32
}

type Collector interface {
	Collect(stats []CPUTimeRecord)
}

type CPUCollector struct {
	ctx       context.Context
	collector Collector
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	started   bool
}

func NewCPUCollector(c Collector) *CPUCollector {
	return &CPUCollector{
		collector: c,
	}
}

var defCollectTickerInterval = time.Second

var defProfileDuration = 500 * time.Millisecond

func (cc *CPUCollector) Start() {
	if cc.started {
		return
	}
	cc.started = true
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	cc.wg.Add(1)
	go cc.collectCPULoop()
}

func (cc *CPUCollector) Stop() {
	if !cc.started {
		return
	}
	cc.started = false
	if cc.cancel != nil {
		cc.cancel()
	}
	cc.wg.Wait()
}

func (cc *CPUCollector) collectCPULoop() {
	profileConsumer := make(chan []byte, 1)
	ticker := time.NewTicker(defCollectTickerInterval)
	defer func() {
		cc.wg.Done()
		ticker.Stop()
	}()

	for {
		select {
		case <-cc.ctx.Done():
			return
		case <-ticker.C:
			data, err := cc.captureCPUProfile()
			if err != nil {
				continue
			}
			if data != nil {
				profileConsumer <- data
			}
		case data := <-profileConsumer:
			cc.handleProfileData(data)
		}
	}
}

func (cc *CPUCollector) captureCPUProfile() ([]byte, error) {
	f, err := os.CreateTemp("", "cpu_profile_*.prof")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())

	if err := pprof.StartCPUProfile(f); err != nil {
		return nil, err
	}
	time.Sleep(defProfileDuration)
	pprof.StopCPUProfile()

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (cc *CPUCollector) handleProfileData(data []byte) {
	err := os.WriteFile("collected_profile.prof", data, 0644)
	if err != nil {
		return
	}
	p, err := profile.ParseData(data)
	if err != nil {
		return
	}
	stats := cc.parseCPUProfileByLabels(p)
	cc.collector.Collect(stats)
}

func (cc *CPUCollector) parseCPUProfileByLabels(p *profile.Profile) []CPUTimeRecord {
	labelMap := make(map[string]int64)
	idx := len(p.SampleType) - 1
	for _, s := range p.Sample {
		labels, ok := s.Label[labelKey]
		if !ok || len(labels) == 0 {
			continue
		}
		for _, label := range labels {
			labelMap[label] += s.Value[idx]
		}
	}
	return cc.createLabelStats(labelMap)
}

func (cc *CPUCollector) createLabelStats(labelMap map[string]int64) []CPUTimeRecord {
	stats := make([]CPUTimeRecord, 0, len(labelMap))
	for label, cpuTime := range labelMap {
		stats = append(stats, CPUTimeRecord{
			Label:     label,
			CPUTimeMs: uint32(time.Duration(cpuTime).Milliseconds()),
		})
	}
	return stats
}
