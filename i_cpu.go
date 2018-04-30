package main

import (
	"os/exec"
	"os"
	"strings"
	"strconv"
	"time"
)

type CPUStats struct {
	ps             PS
	PerCPU         bool `toml:"percpu"`
	TotalCPU       bool `toml:"totalcpu"`
	CollectCPUTime bool `toml:"collect_cpu_time"`
	ReportActive   bool `toml:"report_active"`
}

func NewCPUStats(ps PS) *CPUStats {
	return &CPUStats{
		ps:             ps,
		CollectCPUTime: true,
		ReportActive:   true,
	}
}

func (_ *CPUStats) Description() string {
	return "Read metrics about cpu usage"
}

var sampleConfig = `
  ## Whether to report per-cpu stats or not
  percpu = true
  ## Whether to report total system cpu stats or not
  totalcpu = true
  ## If true, collect raw CPU time metrics.
  collect_cpu_time = false
  ## If true, compute and report the sum of all non-idle CPU states.
  report_active = false
`

func (_ *CPUStats) SampleConfig() string {
	return sampleConfig
}

func (s *CPUStats) Gather(acc Accumulator) error {
	output, err := exec.Command("vmstat", "-S").CombinedOutput()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}

	now := time.Now()

	stats := string(output)
	rows := strings.Split(stats, "\n")
	rows = rows[1:]
	data := make(map[string]float64)
	headers := strings.Fields(rows[0])
	values := strings.Fields(rows[1])
	for count := 0; count < len(headers); count++ {
		v, _ := strconv.ParseFloat(values[count], 64)
		data[headers[count]] = v
	}

	tags := map[string]string{
		"cpu": "cpu-total",
	}

	fieldsC := map[string]interface{}{
		"usage_idle":   data["id"],
		"usage_system": data["sy"],
		"usage_user":   data["us"],
	}

	acc.AddCounter("cpu", fieldsC, tags, now)
	return nil
}
