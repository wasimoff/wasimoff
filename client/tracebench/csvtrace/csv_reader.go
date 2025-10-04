package csvtrace

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
)

// Huawei 2023 private dataset: https://github.com/sir-lab/data-release/
// The CSV files all have the same format, where each cell means something different
// though:  day,time,0,1,2,3,4,5,[...],195,196,197,198,199
// Day increments in each file, time is a continious second range from 0 to 20303940.
// Some floats need float64 to be represented exactly but float32 should suffice.
// Note that there are "holes" in the days: there are 141 files spanning 234 days.

type HuaweiDataset struct {
	// request rate for replaying
	RequestsPerMinute ColumnSet
	// average length of each function invocation
	FunctionDelayAvgPerMinute ColumnSet
}

// Read all datafiles belonging to a given day and return as frames.
func ReadDataset(directory string, cols []string) HuaweiDataset {

	load := func(dataset string) ColumnSet {
		// construct filename and read csv file
		filename := path.Join(directory, fmt.Sprintf("%s.csv.gz", dataset))
		log.Printf("loading %s ...", filename)
		cs, err := ReadCSV(filename, cols)
		if err != nil {
			log.Fatalf("failed reading csv: %v", err)
		}
		return cs
	}

	return HuaweiDataset{
		RequestsPerMinute:         load("requests_minute"),
		FunctionDelayAvgPerMinute: load("function_delay_minute"),
	}

}

// Scale all columns in the dataset to simplify downstream usage.
func (hw *HuaweiDataset) ScaleDatasets(rateScale float64, tasklenScale float64) *HuaweiDataset {

	scaleColumnSet := func(cs ColumnSet, mult float64) {
		for key := range cs {
			if key == "time" {
				continue
			}
			for i, v := range cs[key] {
				cs[key][i] = v * mult
			}
		}
	}

	if rateScale != 1.0 {
		log.Printf("scaling the RequestsPerMinute values (y-axis) by %f", rateScale)
		scaleColumnSet(hw.RequestsPerMinute, rateScale)
	}
	if tasklenScale != 1.0 {
		log.Printf("scaling the FunctionDelayAvgPerMinute values (y-axis) by %f", tasklenScale)
		scaleColumnSet(hw.FunctionDelayAvgPerMinute, tasklenScale)
	}

	return hw

}

// Holds the parsed CSV data in a set of column slices..
type ColumnSet map[string][]float64

func ReadCSV(filename string, keep []string) (ColumnSet, error) {

	// validate the column names "to keep"
	for _, col := range keep {
		i, err := strconv.Atoi(col)
		if err != nil || i < 0 || i >= 200 {
			return nil, fmt.Errorf("not a valid data column: %s", col)
		}
	}

	// open the file
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open %q: %s", filename, err)
	}
	defer f.Close()

	// use it as a file interface
	var file io.Reader = f

	// maybe it needs decompression
	if strings.HasSuffix(filename, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("failed to open %q as gzip: %s", filename, err)
		}
		defer gz.Close()
		file = gz
	}

	// open a csv reader
	reader := csv.NewReader(file)

	// read header and make sure there is a time column
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	if header[1] != "time" {
		return nil, fmt.Errorf("unexpected header: wanted 'time' in second column")
	}

	// store the indices of the columns we want to keep
	indices := make(map[string]int)
	indices["time"] = 1
	for _, col := range keep {
		i := slices.Index(header, col)
		if i == -1 {
			return nil, fmt.Errorf("unexpected header: column %q not found", col)
		}
		indices[col] = i
	}

	// initialize the result set and allocate slices
	dataset := make(ColumnSet)
	dataset["time"] = make([]float64, 0)
	for col := range indices {
		dataset[col] = make([]float64, 0)
	}

	// read data rows from file
	for row := 0; ; row++ {

		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("reading row %d: %w", row, err)
		}

		for col, i := range indices {
			value, err := parseFloat(record[i])
			if err != nil {
				return nil, fmt.Errorf("parsing row %d: column %s (%d): %w", row, col, i, err)
			}
			dataset[col] = append(dataset[col], value)
		}

	}

	return dataset, nil

}

// parseFloat parses a string as float64, treating empty strings as 0
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}
