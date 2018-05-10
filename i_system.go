package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"os/exec"
	"strconv"
	"time"
	"runtime"
	"log"
)

type SystemStats struct{}

func (_ *SystemStats) Description() string {
	return "Read metrics about system load & uptime"
}

func (_ *SystemStats) SampleConfig() string { return "" }

func (_ *SystemStats) Gather(acc Accumulator) error {
	output, err := exec.Command("uptime").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error getting System info: %s", err.Error())
	}

	stats := string(output)
	log.Printf("D! Uptime Response: %s\n", stats)
	rows := strings.Split(stats, "\n")

	//values := strings.Fields(rows[0])
	uptimeOutput := strings.Split(rows[0], ",")

	load1, err := strconv.ParseFloat(strings.Trim(uptimeOutput[3], "load average: "), 64)
	load5, err := strconv.ParseFloat(strings.Trim(uptimeOutput[4], " "), 64)
	load15, err := strconv.ParseFloat(strings.Trim(uptimeOutput[5], " "), 64)
	users, err := strconv.ParseUint(strings.Trim(uptimeOutput[2], " users"), 10, 64)
	uptime, err := Uptime()


	acc.AddGauge("system", map[string]interface{}{
		"load1":   load1,
		"load5":   load5,
		"load15":  load15,
		"n_users": users,
		"n_cpus":  runtime.NumCPU(),
	}, nil)
	acc.AddCounter("system", map[string]interface{}{
		"uptime": uptime,
	}, nil)
	acc.AddFields("system", map[string]interface{}{
		"uptime_format": format_uptime(uptime),
	}, nil)

	return nil
}

func BootTime() (uint64, error) {
	kstat, err := exec.LookPath("/usr/bin/kstat")
	if err != nil {
		return 0, err
	}

	out, err := exec.Command(kstat, "-p", "unix:0:system_misc:boot_time").CombinedOutput()
	if err != nil {
		return 0, err
	}

	output := string(out)

	kstats := strings.Fields(output)
	if len(kstats) != 2 {
		return 0, fmt.Errorf("expected 2 kstat, found %d", len(kstats))
	}

	return strconv.ParseUint(kstats[1], 10, 64)
}

func Uptime() (uint64, error) {
	bootTime, err := BootTime()
	if err != nil {
		return 0, err
	}
	return uptimeSince(bootTime), nil
}

func uptimeSince(since uint64) uint64 {
	return uint64(time.Now().Unix()) - since
}

func format_uptime(uptime uint64) string {
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)

	days := uptime / (60 * 60 * 24)

	if days != 0 {
		s := ""
		if days > 1 {
			s = "s"
		}
		fmt.Fprintf(w, "%d day%s, ", days, s)
	}

	minutes := uptime / 60
	hours := minutes / 60
	hours %= 24
	minutes %= 60

	fmt.Fprintf(w, "%2d:%02d", hours, minutes)

	w.Flush()
	return buf.String()
}
