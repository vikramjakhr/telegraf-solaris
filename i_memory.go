package main

import (
	"regexp"
	"os/exec"
	"strconv"
	"errors"
	"time"
	"strings"
	"fmt"
)

type MemStats struct {
	ps PS
}

func (_ *MemStats) Description() string {
	return "Read metrics about memory usage"
}

func (_ *MemStats) SampleConfig() string { return "" }

var globalZoneMemoryCapacityMatch = regexp.MustCompile(`Memory size: ([\d]+) Megabytes`)

func globalZoneMemoryCapacity() (uint64, error) {
	prtconf, err := exec.LookPath("/usr/sbin/prtconf")
	if err != nil {
		return 0, err
	}

	out, err := exec.Command(prtconf).CombinedOutput()
	if err != nil {
		return 0, err
	}

	match := globalZoneMemoryCapacityMatch.FindAllStringSubmatch(string(out), -1)
	if len(match) != 1 {
		return 0, errors.New("memory size not contained in output of /usr/sbin/prtconf")
	}

	totalMB, err := strconv.ParseUint(match[0][1], 10, 64)
	if err != nil {
		return 0, err
	}

	return totalMB * 1024 * 1024, nil
}

func (s *MemStats) Gather(acc Accumulator) error {
	now := time.Now()

	total, err := globalZoneMemoryCapacity()
	if err != nil {
		return err
	}

	// Memory
	output, err := exec.Command("/usr/bin/vmstat", "-S", "1", "2").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error getting Memory info: %s", err.Error())
	}

	stats := string(output)
	rows := strings.Split(stats, "\n")
	rows = rows[1:]
	data := make(map[string]uint64)
	headers := strings.Fields(rows[0])
	values := strings.Fields(rows[2])
	for count := 0; count < len(headers); count++ {
		v, _ := strconv.ParseUint(values[count], 10, 0)
		data[headers[count]] = v
	}

	free := data["free"] * 1024

	fields := map[string]interface{}{
		"total":             total,
		"available":         free,
		"free":              free,
		"used":              total - free,
		"available_percent": 100 * float64(free) / float64(total),
		"used_percent":      100 * float64(total-free) / float64(total),
	}

	acc.AddCounter("mem", fields, nil, now)

	// Paging
	pagingData := make(map[string]float64)
	for count := 0; count < len(headers); count++ {
		v, _ := strconv.ParseFloat(values[count], 64)
		pagingData[headers[count]] = v
	}

	fields = map[string]interface{}{
		"pgpgin_per_s":  pagingData["pi"] * 1024,
		"pgpgout_per_s": pagingData["po"] * 1024,
		"pgfree_per_s":  pagingData["fr"] * 1024,
		"pgscand_per_s": pagingData["sr"],
		"fault_per_s":   pagingData["in"],
	}

	acc.AddCounter("paging", fields, nil, now)
	return nil
}
