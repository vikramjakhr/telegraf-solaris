package main

import (
	"os/exec"
	"strings"
	"strconv"
	"time"
	"fmt"
)

type DiskStats struct {
	ps PS

	// Legacy support
	Mountpoints []string

	MountPoints []string
	IgnoreFS    []string `toml:"ignore_fs"`
}

func (_ *DiskStats) Description() string {
	return "Read metrics about disk usage by mount point"
}

var diskSampleConfig = `
  ## By default, telegraf gather stats for all mountpoints.
  ## Setting mountpoints will restrict the stats to the specified mountpoints.
  # mount_points = ["/"]

  ## Ignore some mountpoints by filesystem type. For example (dev)tmpfs (usually
  ## present on /run, /var/run, /dev/shm or /dev).
  ignore_fs = ["tmpfs", "devtmpfs", "devfs"]
`

func (_ *DiskStats) SampleConfig() string {
	return diskSampleConfig
}

func (s *DiskStats) Gather(acc Accumulator) error {
	output, err := exec.Command("/usr/sbin/df", "-k").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error getting Disk info: %s", err.Error())
	}

	now := time.Now()

	stats := string(output)
	rows := strings.Split(stats, "\n")
	rows = rows[1:len(rows)-1]
	for _, row := range rows {
		data := strings.Fields(row)
		tags := map[string]string{
			"path":   data[5],
			"device": data[0],
		}
		total, _ := strconv.ParseUint(data[1], 10, 0)
		total = total * 1024
		used, _ := strconv.ParseUint(data[2], 10, 0)
		used = used * 1024
		free, _ := strconv.ParseUint(data[3], 10, 0)
		free = free * 1024

		var used_percent float64
		if used+free > 0 {
			used_percent = float64(used) /
				(float64(used) + float64(free)) * 100
		}

		fields := map[string]interface{}{
			"total":        total,
			"used":         used,
			"free":         free,
			"used_percent": used_percent,
		}

		acc.AddGauge("disk", fields, tags, now)
	}

	output, err = exec.Command("/usr/bin/iostat", "-x", "1", "2").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error getting Disk info: %s", err.Error())
	}

	stats = string(output)
	rows = strings.Split(stats, "\n")
	rows = rows[:len(rows)-1]
	for count := len(rows) - 1; ; count-- {
		data := strings.Fields(rows[count])
		if data[0] != "device" && data[1] != "r/s" {
			tags := map[string]string{
				"device": data[0],
			}
			r_s, _ := strconv.ParseFloat(data[1], 64)
			w_s, _ := strconv.ParseFloat(data[2], 64)
			wait, _ := strconv.ParseFloat(data[5], 64)
			svc_t, _ := strconv.ParseFloat(data[7], 64)
			b, _ := strconv.ParseFloat(data[9], 64)

			fields := map[string]interface{}{
				"rd_sec_per_s": r_s,
				"wr_sec_per_s": w_s,
				"await":        wait,
				"tps":          svc_t,
				"avgqu-sz":     b,
			}

			acc.AddGauge("disk", fields, tags, now)
		} else {
			break
		}
	}

	return nil
}
