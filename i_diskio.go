package main

import (
	"os/exec"
	"strings"
	"strconv"
	"time"
	"fmt"
	"regexp"
)

type DiskIOStats struct {
	Devices []string
}

func (_ *DiskIOStats) Description() string {
	return "Read metrics about disk IO by device"
}

var diskIoSampleConfig = `
  ## By default, telegraf will gather stats for all devices including
  ## disk partitions.
  ## Setting devices will restrict the stats to the specified devices.
  # devices = ["sda", "sdb"]
  ## Uncomment the following line if you need disk serial numbers.
  # skip_serial_number = false
  #
  ## On systems which support it, device metadata can be added in the form of
  ## tags.
  ## Currently only Linux is supported via udev properties. You can view
  ## available properties for a device by running:
  ## 'udevadm info -q property -n /dev/sda'
  # device_tags = ["ID_FS_TYPE", "ID_FS_USAGE"]
  #
  ## Using the same metadata source as device_tags, you can also customize the
  ## name of the device via templates.
  ## The 'name_templates' parameter is a list of templates to try and apply to
  ## the device. The template may contain variables in the form of '$PROPERTY' or
  ## '${PROPERTY}'. The first template which does not contain any variables not
  ## present for the device is used as the device name tag.
  ## The typical use case is for LVM volumes, to get the VG/LV name instead of
  ## the near-meaningless DM-0 name.
  # name_templates = ["$ID_FS_LABEL","$DM_VG_NAME/$DM_LV_NAME"]
`

func (_ *DiskIOStats) SampleConfig() string {
	return diskIoSampleConfig
}

func (s *DiskIOStats) Gather(acc Accumulator) error {

	var devices []string

	if len(s.Devices) > 0 {
		devices = s.Devices
	} else {
		output, err := exec.Command("iostat", "-d").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error getting DiskIO info: %s", err.Error())
		}
		devices = strings.Fields(strings.Split(string(output), "\n")[0])
	}

	for _, device := range devices {
		output, err := exec.Command("kstat", "-p", fmt.Sprintf("*:*:%s:*", device)).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error getting DiskIO (kstat) info: %s", err.Error())
		}
		s := string(output)
		if s != "" {
			fields := map[string]interface{}{}
			tags := map[string]string{
				"name": device,
			}
			stats := strings.Split(s, "\n")
			stats = stats[0: len(stats)-1]
			for _, row := range stats {
				data := strings.Fields(row)
				reg := regexp.MustCompile(".*:.*:.*:")
				field := reg.ReplaceAllString(data[0], "${1}")

				switch field {
				case "reads":
					fields["reads"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "writes":
					fields["writes"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "rtime":
					f, _ := strconv.ParseFloat(data[1], 64)
					fields["read_time"] = int(f)
					break
				case "wtime":
					f, _ := strconv.ParseFloat(data[1], 64)
					fields["write_time"] = int(f)
					break
				case "nread":
					fields["read_bytes"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "nwritten":
					fields["write_bytes"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "rcnt":
					fields["iops_in_progress"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				case "wcnt":
					fields["iops_in_progress"], _ = strconv.ParseInt(data[1], 10, 0)
					break
				}
			}
			acc.AddGauge("diskio", fields, tags, time.Now())
		}
	}
	return nil
}