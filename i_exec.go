package main

import (
	"path/filepath"
	"strings"
	"sync"
	"time"
	"os/exec"
	"regexp"
	"strconv"
	"bytes"
	"log"
)

const sampleExecConfig = `
  ## Commands array
  commands = [
    "/tmp/test.sh",
    "/usr/bin/mycollector --foo=bar",
    "/tmp/collect_*.sh"
  ]

  ## Timeout for each command to complete.
  timeout = "5s"
`

type Exec struct {
	Commands []string
	Timeout  Duration
}

func NewExec() *Exec {
	return &Exec{
		Timeout: Duration{Duration: time.Second * 5},
	}
}

func (e *Exec) ProcessCommand(command string, acc Accumulator, wg *sync.WaitGroup) {
	defer wg.Done()
	arr := strings.Split(command, " ")
	if len(arr) > 0 {
		var (
			out    bytes.Buffer
			stderr bytes.Buffer
		)
		var execCmd *exec.Cmd
		if len(arr) > 1 {
			execCmd = exec.Command(arr[0], arr[1:]...)
		} else {
			execCmd = exec.Command(arr[0])
		}
		execCmd.Stdout = &out
		execCmd.Stderr = &stderr

		if err := RunTimeout(execCmd, e.Timeout.Duration); err != nil {
			acc.AddError(err)
		}
		now := time.Now()

		execOutput := strings.Trim(out.String(), "\n")
		if execOutput != "" {
			lines := strings.Split(execOutput, "\n")
			for _, line := range lines {
				data := strings.SplitN(line, " ", 2)
				if len(data) == 2 {
					measurement := parseMeasurement(data[0])
					fields := parseFields(data[1])
					tags := parseTags(data[0])
					if measurement != "" && fields != nil && len(fields) != 0 && tags != nil && len(tags) != 0 {
						acc.AddFields(measurement, parseFields(data[1]), parseTags(data[0]), now)
					}
				}
			}
		}
	}
}

func parseMeasurement(str string) string {
	return strings.Trim(strings.SplitN(str, ",", 2)[0], " ")
}

func parseTags(str string) map[string]string {
	if str == "" {
		return nil
	}
	str = strings.SplitN(str, ",", 2)[1]
	tags := make(map[string]string)
	arr := strings.Split(str, ",")
	for _, value := range arr {
		data := strings.Split(value, "=")
		if len(data) == 2 && data[1] != "" {
			tags[data[0]] = data[1]
		} else {
			log.Printf("E! Error while parsing tag %s", data[0])
			return nil
		}
	}
	return tags
}

func parseFields(str string) map[string]interface{} {
	if str == "" {
		return nil
	}
	fields := make(map[string]interface{})
	arr := strings.Split(str, ",")
	for _, value := range arr {
		data := strings.Split(value, "=")
		if len(data) == 2 && data[1] != ""{
			if isInteger(data[1]) {
				val, err := strconv.Atoi(strings.TrimSuffix(data[1], "i"))
				if err == nil {
					fields[data[0]] = val
				}
			} else {
				val, err := convertToFloat(data[1])
				if err != nil {
					fields[data[0]] = data[1]
				} else {
					fields[data[0]] = val
				}
			}
		} else {
			log.Printf("E! Error while parsing field %s", data[0])
			return nil
		}
	}
	return fields
}

func isInteger(str string) bool {
	match, _ := regexp.MatchString("([0-9]+i)", str)
	return match
}

func convertToFloat(str string) (float64, error) {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (e *Exec) SampleConfig() string {
	return sampleExecConfig
}

func (e *Exec) Description() string {
	return "Read metrics from one or more commands that can output to stdout"
}

func (e *Exec) Gather(acc Accumulator) error {
	var wg sync.WaitGroup
	// Legacy single command support

	commands := make([]string, 0, len(e.Commands))
	for _, pattern := range e.Commands {
		cmdAndArgs := strings.SplitN(pattern, " ", 2)
		if len(cmdAndArgs) == 0 {
			continue
		}

		matches, err := filepath.Glob(cmdAndArgs[0])
		if err != nil {
			acc.AddError(err)
			continue
		}

		if len(matches) == 0 {
			// There were no matches with the glob pattern, so let's assume
			// that the command is in PATH and just run it as it is
			commands = append(commands, pattern)
		} else {
			// There were matches, so we'll append each match together with
			// the arguments to the commands slice
			for _, match := range matches {
				if len(cmdAndArgs) == 1 {
					commands = append(commands, match)
				} else {
					commands = append(commands,
						strings.Join([]string{match, cmdAndArgs[1]}, " "))
				}
			}
		}
	}

	wg.Add(len(commands))
	for _, command := range commands {
		go e.ProcessCommand(command, acc, &wg)
	}
	wg.Wait()
	return nil
}
