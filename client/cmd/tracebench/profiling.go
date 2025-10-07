package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

func StartProfiling(basename string) (stop func(), err error) {

	fCpuProfile := basename + ".cpu.pprof"
	fMemProfile := basename + ".mem.pprof"

	cpu, err := os.OpenFile(fCpuProfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("profiling: cannot open cpu profile %q: %v", fCpuProfile, err)
	}

	mem, err := os.OpenFile(fMemProfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("profiling: cannot open mem profile %q: %v", fMemProfile, err)
	}

	err = pprof.StartCPUProfile(cpu)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "\033[33m// Started pprof profiling. Use 'go tool pprof' to show:\n//  go tool pprof -http localhost:8000 -no_browser %q\033[0m\n", fCpuProfile)

	// should be called in defer at the end of main
	stop = func() {
		// stop cpu profiler
		pprof.StopCPUProfile()
		// get up-to-date statistics
		runtime.GC()
		// write a profile similar to go test -memprofile with allocations
		if err := pprof.Lookup("allocs").WriteTo(mem, 0); err != nil {
			// end of main, not much we can do now
			log.Printf("ERR: failed writing memory profile: %v", err)
		}
	}

	return stop, nil

}
