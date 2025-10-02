package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Args struct {
	Dataset      string
	Broker       string
	Tracefile    string
	DryRun       bool
	ScaleRate    float64
	ScaleTasklen float64
	Columns      []string
}

func cmdline() Args {

	dataset := flag.String("dataset", getEnv("DATASET", "./dataset"),
		"directory with the {requests,function_delay}_minute.csv.gz files")

	broker := flag.String("broker", getEnv("BROKER", "http://localhost:4080"),
		"connection URL to the Wasimoff Broker or ArtDeco client")

	tracefile := flag.String("trace", getEnv("FILE", ""),
		"output file for JSONL formatted task traces")

	dryrun := flag.Bool("dryrun", getBool("dryrun", false),
		"local dry-run without actually sending any tasks to Broker")

	scaleRate := flag.Float64("scale-rate", getFloat("SCALE_RATE", 1.0),
		"global modifier for the request rate (e.g. to slow down)")

	scaleTasklen := flag.Float64("scale-tasklen", getFloat("SCALE_TASKLEN", 1.0),
		"global modifier for the task length (e.g. for shorter tasks)")

	// parse remaining args as list of column indices
	flag.Parse()
	columns := flag.Args()
	if len(columns) == 0 {
		fatal("need at least one column index")
	}
	for _, col := range columns {
		if v, err := strconv.Atoi(col); err == nil && 0 <= v && v <= 200 {
			continue
		} else {
			fatal("cannot parse %v as column index (0..200)", col)
		}
	}

	return Args{
		Dataset:      *dataset,
		Broker:       *broker,
		Tracefile:    *tracefile,
		DryRun:       *dryrun,
		ScaleRate:    *scaleRate,
		ScaleTasklen: *scaleTasklen,
		Columns:      columns,
	}

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
