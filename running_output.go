package main

import (
	"sync"
)

const (
	// Default size of metrics batch size.
	DEFAULT_METRIC_BATCH_SIZE = 1000

	// Default number of metrics kept. It should be a multiple of batch size.
	DEFAULT_METRIC_BUFFER_LIMIT = 10000
)

// RunningOutput contains the output configuration
type RunningOutput struct {
	Name   string
	Output Output
	Config *OutputConfig

	// Guards against concurrent calls to the Output as described in #3009
	sync.Mutex
}

func NewRunningOutput(
	name string,
	output Output,
	conf *OutputConfig,
) *RunningOutput {
	ro := &RunningOutput{
		Name:   name,
		Output: output,
		Config: conf,
	}
	return ro
}

// TODO
func (ro *RunningOutput) write() error {
	return nil
}

// OutputConfig containing name and filter
type OutputConfig struct {
	Name string
}
