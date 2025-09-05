package main

import (
	"compress/gzip"
	"io"
	"log"
	"os"
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

	return qframe.ReadCSV(reader,
		qcsv.EmptyNull(true),
		allYourColumnsAreFloat(),
	)

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
