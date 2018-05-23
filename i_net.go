package main

import (
	"regexp"
	"fmt"
	"os/exec"
	"strings"
	"strconv"
	"time"
	"log"
)

type NetIOStats struct {
	Interfaces []string
}

func (_ *NetIOStats) Description() string {
	return "Read metrics about network interface usage"
}

var netSampleConfig = `
  ## By default, telegraf gathers stats from any up interface (excluding loopback)
  ## Setting interfaces will tell it to gather these explicit interfaces,
  ## regardless of status.
  ##
  # interfaces = ["eth0"]
`

func (_ *NetIOStats) SampleConfig() string {
	return netSampleConfig
}

func (s *NetIOStats) Gather(acc Accumulator) error {
	interfaces := map[string]string{}

	if len(s.Interfaces) > 0 {
		for _, value := range s.Interfaces {
			interfaces[value] = ""
		}
	} else {
		c1, err := exec.Command("ifconfig", "-a").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error getting NetIOStat info: %s", err.Error())
		}

		rows := strings.Split(string(c1), "\n")
		rows = rows[0:len(rows)-1]

		for _, row := range rows {
			idx := strings.Index(row, ": ")
			if idx > 0 {
				interfaces[row[0:idx]] = ""
			}
		}
	}

	for inet, _ := range interfaces {
		output, err := exec.Command("kstat", "-p", fmt.Sprintf("::%s", inet)).CombinedOutput()
		if err != nil {
			log.Printf("D! Error getting NetIO (kstat) info: %s\n", err.Error())
			continue
		}
		s := string(output)
		if s != "" {
			fields := map[string]interface{}{}
			tags := map[string]string{
				"interface": inet,
			}
			stats := strings.Split(s, "\n")
			stats = stats[0: len(stats)-1]
			for _, row := range stats {
				data := strings.Fields(row)
				reg := regexp.MustCompile(".*:.*:.*:")
				field := reg.ReplaceAllString(data[0], "${1}")

				switch field {
				case "obytes":
					fields["bytes_sent"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "rbytes":
					fields["bytes_recv"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "opackets":
					fields["packets_sent"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "ipackets":
					fields["packets_recv"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "ierrors":
					fields["err_in"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "oerrors":
					fields["err_out"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				}
			}
			acc.AddGauge("net", fields, tags, time.Now())
		}
	}
	return nil
}
