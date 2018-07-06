package main

import (
	"os/exec"
	"strings"
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
					fields["result_type"] = 0
					fields["cpu_usage"] = data[0]
					fields["memory_rss"] = data[1]
					fields["memory_vms"] = data[2]
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
		command = exec.Command("pgrep", "-f", pattern, "-u", user)
	}
	out, err := command.Output()
	if err != nil {
		return nil, err
	}

	return out, err
}

func processPID(pid string) ([]byte, error) {
	out, err := exec.Command("ps", "-p", pid, "-o", "pcpu=", "-o", "rss=", "-o", "vsz=").Output()
	if err != nil {
		return nil, err
	}

	return out, err
}
