package cpuprofile

import (
	"CPUProfiler/util"
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"
)

var DefProfileDuration = time.Second

var globalCPUProfiler = newParallelCPUProfiler()

type ProfileConsumer = chan *ProfileData

type ProfileData struct {
	Data  *bytes.Buffer
	Error error
}

func StartCPUProfiler() error {
	return globalCPUProfiler.start()
}

func StopCPUProfiler() {
	globalCPUProfiler.stop()
}

func Register(ch ProfileConsumer) {
	globalCPUProfiler.register(ch)
}

func Unregister(ch ProfileConsumer) {
	globalCPUProfiler.unregister(ch)
}

type parallelCPUProfiler struct {
	ctx            context.Context
	cs             map[ProfileConsumer]struct{}
	notifyRegister chan struct{}
	profileData    *ProfileData
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	lastDataSize   int
	sync.Mutex
	started bool
}

func newParallelCPUProfiler() *parallelCPUProfiler {
	return &parallelCPUProfiler{
		cs:             make(map[ProfileConsumer]struct{}),
		notifyRegister: make(chan struct{}),
	}
}

var (
	errProfilerAlreadyStarted = errors.New("parallelCPUProfiler is already started")
)

func (p *parallelCPUProfiler) start() error {
	p.Lock()
	if p.started {
		p.Unlock()
		return errProfilerAlreadyStarted
	}

	p.started = true
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.Unlock()
	p.wg.Add(1)
	go util.WithRecovery(p.profilingLoop, nil)

	fmt.Println("parallel cpu profiler started")
	return nil
}

func (p *parallelCPUProfiler) stop() {
	p.Lock()
	if !p.started {
		p.Unlock()
		return
	}
	p.started = false
	if p.cancel != nil {
		p.cancel()
	}
	p.Unlock()

	p.wg.Wait()
	fmt.Println("parallel cpu profiler stopped")
}

func (p *parallelCPUProfiler) register(ch ProfileConsumer) {
	if ch == nil {
		return
	}
	p.Lock()
	p.cs[ch] = struct{}{}
	p.Unlock()

	select {
	case p.notifyRegister <- struct{}{}:
	default:
	}
}

func (p *parallelCPUProfiler) unregister(ch ProfileConsumer) {
	if ch == nil {
		return
	}
	p.Lock()
	delete(p.cs, ch)
	p.Unlock()
}

func (p *parallelCPUProfiler) profilingLoop() {
	checkTicker := time.NewTicker(DefProfileDuration)
	defer func() {
		checkTicker.Stop()
		pprof.StopCPUProfile()
		p.wg.Done()
	}()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.notifyRegister:
			// If already in profiling, don't do anything.
			if p.profileData != nil {
				continue
			}
		case <-checkTicker.C:
		}
		p.doProfiling()
	}
}

func (p *parallelCPUProfiler) doProfiling() {
	if p.profileData != nil {
		pprof.StopCPUProfile()
		p.lastDataSize = p.profileData.Data.Len()
		p.sendToConsumers()
	}

	if p.consumersCount() == 0 {
		return
	}

	capacity := (p.lastDataSize/4096 + 1) * 4096
	p.profileData = &ProfileData{Data: bytes.NewBuffer(make([]byte, 0, capacity))}
	err := pprof.StartCPUProfile(p.profileData.Data)
	if err != nil {
		p.profileData.Error = err
		// notify error as soon as possible
		p.sendToConsumers()
		return
	}
}

func (p *parallelCPUProfiler) consumersCount() int {
	p.Lock()
	n := len(p.cs)
	p.Unlock()
	return n
}

func (p *parallelCPUProfiler) sendToConsumers() {
	p.Lock()
	defer func() {
		p.Unlock()
		if r := recover(); r != nil {
			fmt.Printf("Recovered panic: %v\n", r)
			fmt.Printf("Stack trace:\n%s\n", debug.Stack())
		}
	}()

	for c := range p.cs {
		select {
		case c <- p.profileData:
		default:
			// ignore
		}
	}
	p.profileData = nil
}
