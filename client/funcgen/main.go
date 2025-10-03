package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func main() {

	if len(os.Args) < 3 {
		log.Fatal("need two arguments: {show|run} <config.yaml>")
	}

	file, err := os.Open(os.Args[2])
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	traceconf := TraceConfig{}
	err = yaml.NewDecoder(file).Decode(&traceconf)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Loaded configuration: %#v\n------------------------------\n", traceconf)

	GlobalSeed = traceconf.Seed
	if GlobalSeed == 0 {
		GlobalSeed = TrueRandom()
	}

	switch os.Args[1] {

	case "show":
		for _, w := range traceconf.Workloads {
			fmt.Println()
			printWorkload(w, 10)
		}
		fmt.Println()

	case "run":
		starter := NewStarter[time.Time]()

		for _, w := range traceconf.Workloads {
			starter.Add(1)
			go runWorkload(w, starter)
		}

		starter.Wait()
		starter.Broadcast(time.Now())

		time.Sleep(time.Until(starter.value.Add(traceconf.Duration)))

	default:
		log.Fatalf("unknown command, expected {show|run}: %s", os.Args[1])

	}

}

func runWorkload(w WorkloadConfig, starter *Starter[time.Time]) {
	for t := range w.TaskTriggers(starter) {
		fmt.Printf("%20s [%3d] elapsed: %9.6f, task: %9.6f\n", w.Name,
			t.Sequence, t.Elapsed.Seconds(), t.Tasklen.Seconds())
	}
}

func printWorkload(w WorkloadConfig, n int) {
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
