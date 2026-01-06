package csvtrace

import (
	"fmt"
	"os"
	"path"
	"time"

	"gopkg.in/yaml.v3"
)

type TraceConfig struct {
	Name     string
	Duration time.Duration
	Dataset  string
	dataset  *HuaweiDataset
	Offset   time.Duration
	Scale    TraceScaling
	Columns  []string
}

type TraceScaling struct {
	Rate    float64
	Tasklen float64
}

func ReadTraceConfig(filename string) (*TraceConfig, error) {

	// open the config file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open file: %w", err)
	}
	defer file.Close()

	// read config as yaml format
	trace := &TraceConfig{}
	err = yaml.NewDecoder(file).Decode(trace)
	if err != nil {
		return nil, fmt.Errorf("decoding yaml: %w", err)
	}

	// make sure we have a name
	if trace.Name == "" {
		return nil, fmt.Errorf("please provide a name in config")
	}

	// make sure we have columns
	if len(trace.Columns) == 0 {
		return nil, fmt.Errorf("must provide columns, can't be empty")
	}

	// join path to dataset and read it
	if !path.IsAbs(trace.Dataset) {
		trace.Dataset = path.Join(path.Dir(filename), trace.Dataset)
	}
	trace.dataset = ReadDataset(trace.Dataset, trace.Columns)

	// make sure scaling is not zero
	s := &trace.Scale
	if s.Rate == 0 {
		s.Rate = 1
	}
	if s.Tasklen == 0 {
		s.Tasklen = 1
	}

	return trace, nil

}

func (t *TraceConfig) GetDataset() *HuaweiDataset {
	return t.dataset
}
