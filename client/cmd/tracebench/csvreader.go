package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	qcsv "github.com/tobgu/qframe/config/csv"
	"github.com/tobgu/qframe/types"

	"github.com/tobgu/qframe"
)

// The CSV files in 2023 data release all have the same format, where each cell means
// something different though:  day,time,0,1,2,3,4,5,[...],195,196,197,198,199
// Day increments in each file, time is a continious second range from 0 to 20303940.
// Some floats need float64 to be represented exactly but float32 should suffice.

// TODO: I wanted to concatenate all days in a directory to have a continious file
// but a) there are "holes" in the days (141 files spanning 234 days) and b) the
// necessary qframe.Append() function is not implemented yet; it just silently
// fails with an empty frame.

type DayTrace struct {

	// request rate for replaying
	RequestsPerSecond qframe.QFrame
	RequestsPerMinute qframe.QFrame

	// average length of each function invocation
	FunctionDelayAvgPerMinute qframe.QFrame
	PlatformDelayAvgPerMinute qframe.QFrame

	// what load was observed on the cluster for comparison
	CPUUsagePerMinute    qframe.QFrame
	MemoryUsagePerMinute qframe.QFrame
	InstancesPerMinute   qframe.QFrame
}

func ReadDay(basedir string, day int) (trace DayTrace) {

	// day files are named the same for all datasets
	filename := fmt.Sprintf("day_%03d.csv.gz", day)
	load := func(dir string) qframe.QFrame {
		log.Printf("loading %s/%s ...", dir, filename)
		return ReadQframe(path.Join(basedir, dir, filename))
	}

	// TODO: maybe we'll need to convert the PerMinute traces to PerSecond manually
	// frame = frame.Apply(
	// 	qframe.Instruction{Fn: func(f float64) float64 { return f / 60 }, SrcCol1: coln, DstCol: coln},
	// )

	// using hardcoded directory names (from ZIP filename) load all files for this day
	return DayTrace{
		RequestsPerSecond:         load("requests_second"),
		RequestsPerMinute:         load("requests_minute"),
		FunctionDelayAvgPerMinute: load("function_delay_minute"),
		PlatformDelayAvgPerMinute: load("platform_delay_minute"),
		CPUUsagePerMinute:         load("cpu_usage_minute"),
		MemoryUsagePerMinute:      load("memory_usage_minute"),
		InstancesPerMinute:        load("instances_minute"),
	}

}

func (t *DayTrace) SelectColumns(cols []string) {
	cols = append([]string{"time"}, cols...)
	t.RequestsPerSecond = t.RequestsPerSecond.Select(cols...)
	t.RequestsPerMinute = t.RequestsPerMinute.Select(cols...)
	t.FunctionDelayAvgPerMinute = t.FunctionDelayAvgPerMinute.Select(cols...)
	t.PlatformDelayAvgPerMinute = t.PlatformDelayAvgPerMinute.Select(cols...)
	t.CPUUsagePerMinute = t.CPUUsagePerMinute.Select(cols...)
	t.MemoryUsagePerMinute = t.MemoryUsagePerMinute.Select(cols...)
	t.InstancesPerMinute = t.InstancesPerMinute.Select(cols...)
}

func ReadQframe(filename string) (frame qframe.QFrame) {

	// open the file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("can't open %q: %s", filename, err)
	}
	defer file.Close()

	// use it as a reader interface
	var reader io.Reader = file

	// maybe it needs decompression
	if strings.HasSuffix(filename, ".gz") {
		zr, err := gzip.NewReader(file)
		if err != nil {
			log.Fatalf("failed to open %q as gzip: %s", filename, err)
		}
		defer zr.Close()
		reader = zr
	}

	frame = qframe.ReadCSV(reader,
		qcsv.EmptyNull(false),
		allYourColumnsAreFloat(),
	)

	// translate all null values to actual zeroes
	return // TODO

}

// set the types map to float64 for all expected column names
func allYourColumnsAreFloat() qcsv.ConfigFunc {
	return func(c *qcsv.Config) {
		c.Types = make(map[string]types.DataType, 202)
		c.Types["day"] = types.Float
		c.Types["time"] = types.Float
		for i := range 200 {
			c.Types[strconv.Itoa(i)] = types.Float
		}
	}
}
