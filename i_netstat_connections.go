package main

import (
	"os/exec"
	"fmt"
	"strings"
)

type NetStatConnections struct {
	// List of status URLs
	Patterns []string
}

func (_ *NetStatConnections) Description() string {
	return "Read TCP metrics such as established, time wait and sockets counts for specified IP and port"
}

var netstatConnectionsSampleConfig = `
	# An array of Nginx stub_status URI to gather stats.
	# patterns = ["10.1.54.119.62720","172.217.26.174:443"]
  	patterns = ["<IP><delimiter><Port>"]
`

func (_ *NetStatConnections) SampleConfig() string {
	return netstatConnectionsSampleConfig
}

func (s *NetStatConnections) Gather(acc Accumulator) error {
	if !s.isValidConfig() {
		return fmt.Errorf("Invalid netstat connection configuration")
	}
	out, err := exec.Command("netstat", "-an").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error executing netstat request")
	}

	netstatOutput := strings.Split(string(out), "\n")

	data := make(map[string]map[string]int64)

	for _, row := range netstatOutput {
		for _, pattern := range s.Patterns {
			var stats map[string]int64
			if data[pattern] != nil {
				stats = data[pattern]
			} else {
				stats = map[string]int64{
					"ESTABLISHED": 0,
					"CLOSE_WAIT":  0,
					"TIME_WAIT":   0,
					"LISTEN":      0,
					"IDLE":        0,
					"SYN_SENT":    0,
					"LAST_ACK":    0,
				}
			}
			if strings.Contains(row, pattern) {
				if strings.Contains(row, "ESTABLISHED") {
					stats["ESTABLISHED"]++
				} else if strings.Contains(row, "CLOSE_WAIT") {
					stats["CLOSE_WAIT"]++
				} else if strings.Contains(row, "TIME_WAIT") {
					stats["TIME_WAIT"]++
				} else if strings.Contains(row, "LISTEN") {
					stats["LISTEN"]++
				} else if strings.Contains(row, "IDLE") {
					stats["IDLE"]++
				} else if strings.Contains(row, "SYN_SENT") {
					stats["SYN_SENT"]++
				} else if strings.Contains(row, "LAST_ACK") {
					stats["LAST_ACK"]++
				}
			}
			data[pattern] = stats
		}
	}

	for pattern, stats := range data {
		fields := make(map[string]interface{})
		for key, value := range stats {
			fields[key] = value
		}
		acc.AddFields("netstat_connections", fields, map[string]string{
			"pattern": pattern,
		})
	}

	return nil
}

func (s *NetStatConnections) isValidConfig() bool {
	if s.Patterns == nil || len(s.Patterns) < 1 {
		return false
	}
	for _, pattern := range s.Patterns {
		split := strings.Split(pattern, ":")
		if len(split) != 2 {
			return false
		}
	}
	return true
}
