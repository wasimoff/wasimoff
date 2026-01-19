package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"wasi.team/client/internal/tracebench/csvtrace"
	"wasi.team/client/internal/tracebench/funcgen"
)

type Args struct {
	RunFuncgen  *funcgen.TraceConfig
	RunCsvTrace *csvtrace.TraceConfig
	Broker      string
	Tracefile   string
	Pprof       bool
	Wait        bool
}

func cmdline() (args *Args, err error) {

	// either one of ...
	funcgenCfg := flag.String("funcgen", getEnv("CONF_FUNCGEN", ""),
		"path to a generated workload config file")
	csvtraceCfg := flag.String("tracer", getEnv("CONF_TRACER", ""),
		"path to a yaml config to follow huawei dataset")

	// URL to the Broker (dry-run when empty)
	broker := flag.String("broker", getEnv("BROKER", ""),
		"connection URL to the Wasimoff Broker or ArtDeco client")

	// log received task traces to this file
	out := fmt.Sprintf("tracebench-%d-%d.pb", time.Now().Unix(), os.Getpid())
	tracefile := flag.String("output", getEnv("OUTPUT", out),
		"output file for delimited protobuf Task_Metadata traces")

	// enable pprof profiler during a run
	pprof := flag.Bool("profile", getBool("PROFILE", false),
		"enable pprof profiling of this binary during the run")

	// wait until all task responses are received
	wait := flag.Bool("wait", getBool("WAITTASKS", false),
		"stop generating after timeout but wait for all task responses")

	// parse arguments and check validity
	flag.Parse()
	args = &Args{Broker: *broker, Tracefile: *tracefile, Pprof: *pprof, Wait: *wait}

	// read one of the config file types
	if (*funcgenCfg == "" && *csvtraceCfg == "") || (*funcgenCfg != "" && *csvtraceCfg != "") {
		return nil, fmt.Errorf("must provide either -funcgen or -tracer config")
	}
	if *funcgenCfg != "" {
		args.RunFuncgen, err = funcgen.ReadTraceConfig(*funcgenCfg)
	}
	if *csvtraceCfg != "" {
		args.RunCsvTrace, err = csvtrace.ReadTraceConfig(*csvtraceCfg)
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	return

}

// read an environment variable or return a default
func getEnv(key, fallback string) string {
	key = prefixedKey(key)
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// read an environment variable as an int or return a default
func getInt(key string, fallback int) int {
	key = prefixedKey(key)
	if value, ok := os.LookupEnv(key); ok {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		} else {
			fatal("cannot parse %v as int: %s", value, err)
		}
	}
	return fallback
}

// read an environment variable as a float or return a default
func getFloat(key string, fallback float64) float64 {
	key = prefixedKey(key)
	if value, ok := os.LookupEnv(key); ok {
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		} else {
			fatal("cannot parse %v as float: %s", value, err)
		}
	}
	return fallback
}

// read an environment variable as a bool or return a default
func getBool(key string, fallback bool) bool {
	key = prefixedKey(key)
	if value, ok := os.LookupEnv(key); ok {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "true", "t", "yes", "y", "on", "1":
			return true
		case "false", "f", "no", "n", "off", "0", "":
			return false
		default:
			fatal("cannot parse %v as bool", value)
		}
	}
	return fallback
}

func prefixedKey(key string) string {
	return "TRACEBENCH_" + strings.ToUpper(key)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERR: "+format+"\n", a...)
	flag.Usage()
	os.Exit(1)
}
