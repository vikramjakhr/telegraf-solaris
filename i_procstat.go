package main

import (
	"os/exec"
	"strings"
	"strconv"
)

type PID int32

type Procstat struct {
	Pattern string
	User    string
}

var procstatSampleConfig = `
  ## pattern as argument for pgrep (ie, pgrep -f <pattern>)
  # pattern = "nginx"
  ## user as argument for pgrep (ie, pgrep -u <user>)
  # user = "nginx"
`

func (_ *Procstat) SampleConfig() string {
	return procstatSampleConfig
}

func (_ *Procstat) Description() string {
	return "Monitor process"
}

func (p *Procstat) Gather(acc Accumulator) error {
	tags := map[string]string{
		"process_name": p.Pattern,
	}

	fields := map[string]interface{}{}

	out, err := execPGrep(p.Pattern, p.User)
	if err != nil {
		fields["result_type"] = 1
		acc.AddFields("procstat", fields, tags)
		return err
	}

	rows := strings.Split(strings.Trim(string(out), " "), "\n")
	rows = rows[0: len(rows)-1]

	if len(rows) > 0 {
		for _, pid := range rows {
			out, err = processPID(pid)
			if err != nil {
				return err
			}

			stats := strings.Trim(string(out), " ")

			if stats != "" {
				data := strings.Fields(stats)
				if len(data) == 3 {
					cu, _ := strconv.ParseFloat(data[0], 64)
					mrss, _ := strconv.ParseInt(data[1], 10, 0)
					mvms, _ := strconv.ParseInt(data[2], 10, 0)
					fields["result_type"] = 0
					fields["cpu_usage"] = cu
					fields["memory_rss"] = mrss
					fields["memory_vms"] = mvms
				}
			} else {
				fields["result_type"] = 1
			}
			acc.AddFields("procstat", fields, tags)
		}
	} else {
		acc.AddFields("procstat", fields, tags)
	}
	return nil
}

func execPGrep(pattern, user string) ([]byte, error) {
	command := exec.Command("/usr/bin/pgrep", "-f", pattern)
	user = strings.Trim(user, " ")
	if user != "" {
		command = exec.Command("/usr/bin/pgrep", "-f", pattern, "-u", user)
	}
	out, err := command.Output()
	if err != nil {
		return nil, err
	}

	return out, err
}

func processPID(pid string) ([]byte, error) {
	out, err := exec.Command("/usr/bin/ps", "-p", pid, "-o", "pcpu=", "-o", "rss=", "-o", "vsz=").Output()
	if err != nil {
		return nil, err
	}

	return out, err
}
