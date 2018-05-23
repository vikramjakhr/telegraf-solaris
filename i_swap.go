package main

import (
	"os/exec"
	"strings"
	"strconv"
	"fmt"
)

type SwapStats struct {
	ps PS
}

func (_ *SwapStats) Description() string {
	return "Read metrics about swap memory usage"
}

func (_ *SwapStats) SampleConfig() string { return "" }

func (s *SwapStats) Gather(acc Accumulator) error {

	output, err := exec.Command("swap", "-s").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error getting Swap info: %s", err.Error())
	}
	sout := string(output)
	if sout != "" {
		swapOutput := strings.Split(sout, "\n")
		swapOutput = swapOutput[0: len(swapOutput)-1]

		s := strings.Split(swapOutput[0], ",")

		if len(s) == 2 {
			kUsed := strings.Trim(strings.Replace(strings.Split(s[0], "=")[1], "k used", "", 1), " ")
			kAvailable := strings.Trim(strings.Replace(s[1], "k available", "", 1), " ")

			used, _ := strconv.ParseUint(kUsed, 10, 0)
			avail, _ := strconv.ParseUint(kAvailable, 10, 0)

			used = used * 1024
			avail = avail * 1024

			total := used + avail
			free := total - used

			var usedPercent float64

			if total != 0 {
				usedPercent = float64(used) / float64(total) * 100.0
			}

			fieldsG := map[string]interface{}{
				"total":        total,
				"used":         used,
				"free":         free,
				"used_percent": usedPercent,
			}

			acc.AddGauge("swap", fieldsG, nil)

			output, err = exec.Command("vmstat", "-S").CombinedOutput()
			if err != nil {
				return fmt.Errorf("error getting Swap Memory info: %s", err.Error())
			}

			vmstats := string(output)
			rows := strings.Split(vmstats, "\n")
			rows = rows[1:]
			data := make(map[string]uint64)
			headers := strings.Fields(rows[0])
			values := strings.Fields(rows[1])
			for count := 0; count < len(headers); count++ {
				v, _ := strconv.ParseUint(values[count], 10, 0)
				data[headers[count]] = v
			}

			fieldsC := map[string]interface{}{
				"in":  data["si"],
				"out": data["so"],
			}

			acc.AddCounter("swap", fieldsC, nil)
		}

	}
	return nil
}
