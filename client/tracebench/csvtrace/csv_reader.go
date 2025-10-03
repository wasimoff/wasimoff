package csvtrace

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

// Huawei 2023 private dataset: https://github.com/sir-lab/data-release/
// The CSV files all have the same format, where each cell means something different
// though:  day,time,0,1,2,3,4,5,[...],195,196,197,198,199
// Day increments in each file, time is a continious second range from 0 to 20303940.
// Some floats need float64 to be represented exactly but float32 should suffice.
// Note that there are "holes" in the days: there are 141 files spanning 234 days.

type HuaweiDataset struct {
	// request rate for replaying
	RequestsPerMinute qframe.QFrame
	// average length of each function invocation
	FunctionDelayAvgPerMinute qframe.QFrame
}

// Read all datafiles belonging to a given day and return as frames.
func ReadDataset(directory string) (trace HuaweiDataset) {

	load := func(dataset string) qframe.QFrame {
		// construct filename and read qframe
		filename := path.Join(directory, fmt.Sprintf("%s.csv.gz", dataset))
		log.Printf("loading %s ...", filename)
		return ReadQframe(filename)
	}

	return HuaweiDataset{
		RequestsPerMinute:         load("requests_minute"),
		FunctionDelayAvgPerMinute: load("function_delay_minute"),
	}

}

// Scale all columns in the dataset to simplify downstream usage.
func (hw *HuaweiDataset) ScaleDatasets(rateScale float64, tasklenScale float64) *HuaweiDataset {

	if rateScale != 1.0 {
		log.Printf("scaling the RequestsPerMinute values (y-axis) by %f", rateScale)
		hw.RequestsPerMinute = scaleEntireQFrame(hw.RequestsPerMinute, rateScale)
	}
	if tasklenScale != 1.0 {
		log.Printf("scaling the FunctionDelayAvgPerMinute values (y-axis) by %f", tasklenScale)
		hw.FunctionDelayAvgPerMinute = scaleEntireQFrame(hw.FunctionDelayAvgPerMinute, tasklenScale)
	}

	return hw

}

func scaleEntireQFrame(frame qframe.QFrame, scale float64) qframe.QFrame {
	for _, column := range frame.ColumnNames() {
		if column == "day" || column == "time" {
			continue
		}
		frame = frame.Apply(qframe.Instruction{Fn: scaleColumn(scale), SrcCol1: column, DstCol: column})
	}
	return frame
}

// Select only specific columns from the datasets.
func (hw *HuaweiDataset) SelectColumns(cols []string) *HuaweiDataset {
	cols = append([]string{"time"}, cols...)
	hw.RequestsPerMinute = hw.RequestsPerMinute.Select(cols...)
	hw.FunctionDelayAvgPerMinute = hw.FunctionDelayAvgPerMinute.Select(cols...)
	return hw
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

	return qframe.ReadCSV(reader, allYourColumnsAreFloat())
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
