package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"wasi.team/client/tracebench"
	"wasi.team/client/tracebench/csvtrace"
	"wasi.team/client/tracebench/funcgen"
)

func main() {

	log.Println("WARNING: this is just a debugging CLI, it is not actually functional in that it doesn't ever send task requests")

	const usage = "{funcgen|csvtrace} config.yaml [run]"

	if len(os.Args) < 3 {
		log.Fatalf("need two arguments: %s", usage)
	}

	run := false
	if len(os.Args) >= 4 && os.Args[3] == "run" {
		run = true
	}

	switch os.Args[1] {

	// generate workload file
	case "funcgen":
		trace, err := funcgen.ReadTraceConfig(os.Args[2])
		if err != nil {
			log.Fatalf("reading funcgen config: %v", err)
		}
		fmt.Printf("Loaded funcgen configuration: %#v\n\n", trace)

		switch run {

		case false:
			for _, w := range trace.Workloads {
				fmt.Println()
				printFuncgen(w, 10)
			}
			fmt.Println()

		case true:
			starter := tracebench.NewStarter[time.Time]()
			for _, w := range trace.Workloads {
				starter.Add(1)
				go runFuncgen(w, starter)
			}
			starter.Wait()
			now := time.Now()
			starter.Broadcast(now)
			time.Sleep(time.Until(now.Add(trace.Duration)))

		}

		// huawei trace dataset
	case "csvtrace":
		trace, err := csvtrace.ReadTraceConfig(os.Args[2])
		if err != nil {
			log.Fatalf("reading csvtrace config: %v", err)
		}
		fmt.Printf("Loaded funcgen configuration: %#v\n\n", trace)

		switch run {

		case false:
			for _, col := range trace.Columns {
				fmt.Println()
				printCSVTrace(trace.GetDataset(), col, 10)
			}
			fmt.Println()

		case true:
			starter := tracebench.NewStarter[time.Time]()
			for _, col := range trace.Columns {
				starter.Add(1)
				go runCSVTrace(trace.GetDataset(), col, starter)
			}
			starter.Wait()
			now := time.Now()
			starter.Broadcast(now)
			time.Sleep(time.Until(now.Add(trace.Duration)))

		}

	default:
		log.Fatalf("unknown command %q: expected: %s", os.Args[1], usage)

	}

}

func runFuncgen(w funcgen.WorkloadConfig, starter *tracebench.Starter[time.Time]) {
	for t := range w.TaskTriggers(starter) {
		fmt.Printf("%20s [%3d] elapsed: %9.6f, task: %9.6f\n", w.Name,
			t.Sequence, t.Elapsed.Seconds(), t.Tasklen.Seconds())
	}
}

func printFuncgen(w funcgen.WorkloadConfig, n int) {
	i := 0
	for t := range w.TaskIterator() {
		i += 1
		fmt.Printf("%20s [%3d] sleep: %9.6f, task: %9.6f, skip: %v\n", w.Name, i,
			t.Sleep.Seconds(), t.Tasklen.Seconds(), t.Skip)
		if i == n {
			break
		}
	}
}

func runCSVTrace(hw *csvtrace.HuaweiDataset, col string, starter *tracebench.Starter[time.Time]) {
	for t := range hw.TaskTriggers(starter, col) {
		fmt.Printf("column %3s [%3d] elapsed: %9.6f, task: %9.6f\n", col,
			t.Sequence, t.Elapsed.Seconds(), t.Tasklen.Seconds())
	}
}

func printCSVTrace(hw *csvtrace.HuaweiDataset, col string, n int) {
	i := 0
	for t := range hw.TaskIterator(col) {
		i += 1
		fmt.Printf("column %3s [%3d] elapsed: %9.6f, task: %9.6f\n", col, i,
			t.Elapsed.Seconds(), t.Tasklen.Seconds())
		if i == n {
			break
		}
	}
}
