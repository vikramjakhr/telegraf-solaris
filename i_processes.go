package main

import (
	"log"
	"os/exec"
	"strings"
)

type Processes struct {
}

func (p *Processes) Description() string {
	return "Get the number of processes and group them by status"
}

func (p *Processes) SampleConfig() string { return "" }

func (p *Processes) Gather(acc Accumulator) error {
	// Get an empty map of metric fields
	fields := getEmptyFields()

	if err := p.gatherFromPS(fields); err != nil {
		return err
	}

	acc.AddFields("processes", fields, nil)
	return nil
}

// TODO: total_threads needs to be implemented
// Gets empty fields of metrics based on the OS
func getEmptyFields() map[string]interface{} {
	fields := map[string]interface{}{
		"blocked":       int64(0),
		"zombies":       int64(0),
		"stopped":       int64(0),
		"running":       int64(0),
		"sleeping":      int64(0),
		"total":         int64(0),
		"unknown":       int64(0),
		"total_threads": int64(0),
		"wait":          int64(0),
	}
	return fields
}

// exec `ps` to get all process states
func (p *Processes) gatherFromPS(fields map[string]interface{}) error {
	out, err := execPS()
	if err != nil {
		return err
	}

	rows := strings.Split(string(out), "\n")

	for _, line := range rows[1:len(rows)-1] {

		stats := strings.Fields(line)

		switch stats[1] {
		case "W":
			fields["wait"] = fields["wait"].(int64) + int64(1)
		case "Z":
			fields["zombies"] = fields["zombies"].(int64) + int64(1)
		case "X":
			fields["dead"] = fields["dead"].(int64) + int64(1)
		case "T":
			fields["stopped"] = fields["stopped"].(int64) + int64(1)
		case "0":
			fields["running"] = fields["running"].(int64) + int64(1)
		case "S":
			fields["sleeping"] = fields["sleeping"].(int64) + int64(1)
		case "I":
			fields["idle"] = fields["idle"].(int64) + int64(1)
		case "?":
			fields["unknown"] = fields["unknown"].(int64) + int64(1)
		default:
			log.Printf("I! processes: Unknown state [ %s ] from ps",
				string(stats[1]))
		}
		fields["total"] = fields["total"].(int64) + int64(1)
	}
	return nil
}

func execPS() ([]byte, error) {
	out, err := exec.Command("ps", "-el").Output()
	if err != nil {
		return nil, err
	}

	return out, err
}
