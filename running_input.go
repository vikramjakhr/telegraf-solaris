package main

import (
	"time"
)

type RunningInput struct {
	Input  Input
	Config *InputConfig

	trace       bool
	defaultTags map[string]string
}

func NewRunningInput(
	input Input,
	config *InputConfig,
) *RunningInput {
	return &RunningInput{
		Input:  input,
		Config: config,
	}
}

// InputConfig containing a name, interval, and filter
type InputConfig struct {
	Name              string
	NameOverride      string
	MeasurementPrefix string
	MeasurementSuffix string
	Tags              map[string]string
	Interval          time.Duration
}

func (r *RunningInput) Name() string {
	return "inputs." + r.Config.Name
}

func (r *RunningInput) Trace() bool {
	return r.trace
}

func (r *RunningInput) SetTrace(trace bool) {
	r.trace = trace
}

func (r *RunningInput) SetDefaultTags(tags map[string]string) {
	r.defaultTags = tags
}
