package main

import (
	"os/exec"
	"strings"
	"fmt"
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
	fields["result_type"] = 0

	out, err := execPGrep(p.Pattern, p.User)
	if err != nil {
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
	command := fmt.Sprintf("pgrep %s", pattern)
	user = strings.Trim(user, " ")
	if user != "" {
		command = fmt.Sprintf("pgrep %s -u %s", pattern, user)
	}
	out, err := exec.Command(command).Output()
	if err != nil {
		return nil, err
	}

	return out, err
}

func processPID(pid string) ([]byte, error) {
	out, err := exec.Command(fmt.Sprintf("ps -p %s  -o pcpu= -o rss= -o vsz=", pid)).Output()
	if err != nil {
		return nil, err
	}

	return out, err
}
