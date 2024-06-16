package cpuprofile

import (
	"bytes"
	"github.com/google/pprof/profile"
	"runtime/pprof"
	"testing"
	"time"
)

func TestBasicAPI(t *testing.T) {
	err := StartCPUProfiler()
	if err != nil {
		t.Fatalf("Failed to start CPU profiler: %v", err)
	}
	defer StopCPUProfiler()

	err = StartCPUProfiler()
	if err != errProfilerAlreadyStarted {
		t.Fatalf("Expected error: %v, got: %v", errProfilerAlreadyStarted, err)
	}

	// Test for close multiple times.
	StopCPUProfiler()
	StopCPUProfiler()

	globalCPUProfiler = newParallelCPUProfiler()
	StopCPUProfiler()
	err = StartCPUProfiler()
	if err != nil {
		t.Fatalf("Failed to start CPU profiler: %v", err)
	}
	err = StartCPUProfiler()
	if err != errProfilerAlreadyStarted {
		t.Fatalf("Expected error: %v, got: %v", errProfilerAlreadyStarted, err)
	}
}

func TestParallelCPUProfiler(t *testing.T) {
	err := StartCPUProfiler()
	if err != nil {
		t.Fatalf("Failed to start CPU profiler: %v", err)
	}
	defer StopCPUProfiler()

	// Test register/unregister nil
	Register(nil)
	if count := globalCPUProfiler.consumersCount(); count != 0 {
		t.Fatalf("Expected 0 consumers, got: %d", count)
	}
	Unregister(nil)
	if count := globalCPUProfiler.consumersCount(); count != 0 {
		t.Fatalf("Expected 0 consumers, got: %d", count)
	}

	// Test profile error and duplicate register.
	dataCh := make(ProfileConsumer, 10)
	err = pprof.StartCPUProfile(bytes.NewBuffer(nil))
	if err != nil {
		t.Fatalf("Failed to start CPU profile: %v", err)
	}

	// Test for duplicate register.
	Register(dataCh)
	Register(dataCh)
	if count := globalCPUProfiler.consumersCount(); count != 1 {
		t.Fatalf("Expected 1 consumer, got: %d", count)
	}

	// Test profile error
	data := <-dataCh
	if data.Error == nil || data.Error.Error() != "cpu profiling already in use" {
		t.Fatalf("Expected error: cpu profiling already in use, got: %v", data.Error)
	}
	Unregister(dataCh)
	if count := globalCPUProfiler.consumersCount(); count != 0 {
		t.Fatalf("Expected 0 consumers, got: %d", count)
	}

	// shouldn't receive data from a unregistered consumer.
	data = nil
	select {
	case data = <-dataCh:
	default:
	}
	if data != nil {
		t.Fatalf("Expected nil data, got: %v", data)
	}

	// unregister not exist consumer
	Unregister(dataCh)
	if count := globalCPUProfiler.consumersCount(); count != 0 {
		t.Fatalf("Expected 0 consumers, got: %d", count)
	}

	// Test register a closed consumer
	dataCh = make(ProfileConsumer, 10)
	close(dataCh)
	Register(dataCh)
	if count := globalCPUProfiler.consumersCount(); count != 1 {
		t.Fatalf("Expected 1 consumer, got: %d", count)
	}
	data, ok := <-dataCh
	if data != nil || ok {
		t.Fatalf("Expected nil data and closed channel, got: %v, %v", data, ok)
	}
	Unregister(dataCh)
	if count := globalCPUProfiler.consumersCount(); count != 0 {
		t.Fatalf("Expected 0 consumers, got: %d", count)
	}
	pprof.StopCPUProfile()

	// Test successfully get profile data.
	dataCh = make(ProfileConsumer, 10)
	Register(dataCh)
	data = <-dataCh
	if data.Error != nil {
		t.Fatalf("Expected no error, got: %v", data.Error)
	}
	profileData, err := profile.ParseData(data.Data.Bytes())
	if err != nil {
		t.Fatalf("Failed to parse profile data: %v", err)
	}
	if profileData == nil {
		t.Fatalf("Expected non-nil profile data")
	}
	Unregister(dataCh)
	if count := globalCPUProfiler.consumersCount(); count != 0 {
		t.Fatalf("Expected 0 consumers, got: %d", count)
	}

	// Test stop profiling when no consumer.
	Register(dataCh)
	for {
		// wait for parallelCPUProfiler do profiling successfully
		err = pprof.StartCPUProfile(bytes.NewBuffer(nil))
		if err != nil {
			break
		}
		pprof.StopCPUProfile()
		time.Sleep(time.Millisecond)
	}
	Unregister(dataCh)
	if count := globalCPUProfiler.consumersCount(); count != 0 {
		t.Fatalf("Expected 0 consumers, got: %d", count)
	}

	// wait for parallelCPUProfiler stop profiling
	start := time.Now()
	for {
		err = pprof.StartCPUProfile(bytes.NewBuffer(nil))
		if err == nil || time.Since(start) >= time.Second*2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Failed to start CPU profile: %v", err)
	}
	pprof.StopCPUProfile()
}
