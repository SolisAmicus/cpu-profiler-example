package collector

import (
	"CPUProfiler/cpuprofile"
	"context"
	"fmt"
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
	ctx        context.Context
	collector  Collector
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	started    bool
	registered bool
}

func NewCPUCollector(c Collector) *CPUCollector {
	return &CPUCollector{
		collector: c,
	}
}

func (cc *CPUCollector) Start() {
	if cc.started {
		return
	}
	cc.started = true
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	cc.wg.Add(1)
	go cc.collectCPULoop()
	fmt.Println("cpu collector started")
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
	fmt.Println("cpu collector stopped")
}

var defCollectTickerInterval = time.Second

func (cc *CPUCollector) collectCPULoop() {
	profileConsumer := make(cpuprofile.ProfileConsumer, 1)
	ticker := time.NewTicker(defCollectTickerInterval)
	defer func() {
		cc.wg.Done()
		cc.doUnregister(profileConsumer)
		ticker.Stop()
	}()

	for {
		cc.doRegister(profileConsumer)
		select {
		case <-cc.ctx.Done():
			return
		case <-ticker.C:
		case data := <-profileConsumer:
			cc.handleProfileData(data)
		}
	}
}

func (cc *CPUCollector) handleProfileData(data *cpuprofile.ProfileData) {
	if data.Error != nil {
		return
	}
	p, err := profile.ParseData(data.Data.Bytes())
	if err != nil {
		return
	}
	stats := cc.parseCPUProfileByLabels(p)
	cc.collector.Collect(stats)
}

func (cc *CPUCollector) doRegister(profileConsumer cpuprofile.ProfileConsumer) {
	if cc.registered {
		return
	}
	cc.registered = true
	cpuprofile.Register(profileConsumer)
}

func (cc *CPUCollector) doUnregister(profileConsumer cpuprofile.ProfileConsumer) {
	if !cc.registered {
		return
	}
	cc.registered = false
	cpuprofile.Unregister(profileConsumer)
}

func (cc *CPUCollector) parseCPUProfileByLabels(p *profile.Profile) []CPUTimeRecord {
	// TODO
}

func (cc *CPUCollector) createLabelStats(labelMap map[string]int64) []CPUTimeRecord {
	// TODO
}

func CtxWithLabel(ctx context.Context, label string) context.Context {
	// TODO
}

func CtxWithLabels(ctx context.Context, labels []string) context.Context {
	// TODO
}
