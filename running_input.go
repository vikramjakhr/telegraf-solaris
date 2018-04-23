package main

import (
	"time"
	"fmt"
)

var GlobalMetricsGathered Stat

type RunningInput struct {
	Input  Input
	Config *InputConfig

	trace       bool
	defaultTags map[string]string

	MetricsGathered Stat
}

func NewRunningInput(
	input Input,
	config *InputConfig,
) *RunningInput {
	return &RunningInput{
		Input:  input,
		Config: config,
		MetricsGathered: Register(
			"gather",
			"metrics_gathered",
			map[string]string{"input": config.Name},
		),
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

// MakeMetric either returns a metric, or returns nil if the metric doesn't
// need to be created (because of filtering, an error, etc.)
func (r *RunningInput) MakeMetric(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	mType ValueType,
	t time.Time,
) Metric {
	m := makemetric(
		measurement,
		fields,
		tags,
		r.Config.NameOverride,
		r.Config.MeasurementPrefix,
		r.Config.MeasurementSuffix,
		r.Config.Tags,
		r.defaultTags,
		true,
		mType,
		t,
	)

	if r.trace && m != nil {
		fmt.Print("> " + m.String())
	}

	r.MetricsGathered.Incr(1)
	GlobalMetricsGathered.Incr(1)
	return m
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
