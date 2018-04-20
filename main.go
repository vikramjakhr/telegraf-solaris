package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof" // Comment this line to disable pprof endpoint.
	"os"
	"log"
	"syscall"
	"os/signal"
	"strings"
)

var fDebug = flag.Bool("debug", false,
	"turn on debug logging")
var fConfig = flag.String("config", "", "configuration file to load")
var fVersion = flag.Bool("version", false, "display the version")
var fSampleConfig = flag.Bool("sample-config", false,
	"print out full sample configuration")
var fInputList = flag.Bool("input-list", false,
	"print available input plugins.")
var fOutputList = flag.Bool("output-list", false,
	"print available output plugins.")
var fUsage = flag.String("usage", "",
	"print usage for a plugin, ie, 'telegraf --usage mysql'")

var (
	nextVersion = "1.5.0"
)

const usage = `Telegraf, The plugin-driven server agent for collecting and reporting metrics.

Usage:

  telegraf [commands|flags]

The commands & flags are:

  config              print out full sample configuration to stdout
  version             print the version to stdout

  --config <file>     configuration file to load
  --test              gather metrics once, print them to stdout, and exit
  --config-directory  directory containing additional *.conf files
  --input-filter      filter the input plugins to enable, separator is :
  --output-filter     filter the output plugins to enable, separator is :
  --usage             print usage for a plugin, ie, 'telegraf --usage mysql'
  --debug             print metrics as they're generated to stdout
  --pprof-addr        pprof address to listen on, format: localhost:6060 or :6060
  --quiet             run in quiet mode

Examples:

  # generate a telegraf config file:
  telegraf config > telegraf.conf

  # generate config with only cpu input & influxdb output plugins defined
  telegraf --input-filter cpu --output-filter influxdb config

  # run a single telegraf collection, outputing metrics to stdout
  telegraf --config telegraf.conf --test

  # run telegraf with all plugins defined in config file
  telegraf --config telegraf.conf

  # run telegraf, enabling the cpu & memory input, and influxdb output plugins
  telegraf --config telegraf.conf --input-filter cpu:mem --output-filter influxdb

  # run telegraf with pprof
  telegraf --config telegraf.conf --pprof-addr localhost:6060
`

func usageExit(rc int) {
	fmt.Println(usage)
	os.Exit(rc)
}

func displayVersion() string {
	return fmt.Sprintf("v%s", nextVersion)
}

func init() {
	AddInput("cpu", func() Input {
		return &CPUStats{
			PerCPU:   true,
			TotalCPU: true,
			ps:       nil,
		}
	})
}

var stop chan struct{}

func main() {
	flag.Usage = func() { usageExit(0) }
	flag.Parse()
	args := flag.Args()

	if len(args) > 0 {
		switch args[0] {
		case "version":
			fmt.Printf("Telegraf %s\n", displayVersion())
			return
		case "config":
			return
		}
	}

	// switch for flags which just do something and exit immediately
	switch {
	case *fOutputList:
		fmt.Println("Available Output Plugins:")
		for k, _ := range Outputs {
			fmt.Printf("  %s\n", k)
		}
		return
	case *fInputList:
		fmt.Println("Available Input Plugins:")
		for k, _ := range Inputs {
			fmt.Printf("  %s\n", k)
		}
		return
	case *fVersion:
		fmt.Printf("Telegraf %s\n", displayVersion())
		return
	case *fSampleConfig:
		return
	case *fUsage != "":
		err := PrintInputConfig(*fUsage)
		err2 := PrintOutputConfig(*fUsage)
		if err != nil && err2 != nil {
			log.Fatalf("E! %s and %s", err, err2)
		}
		return
	}

	stop = make(chan struct{})
	reloadLoop(stop)

}

func reloadLoop(
	stop chan struct{},
) {
	reload := make(chan bool, 1)
	reload <- true
	fmt.Println("starteddddddddddddddddddd")
	for <-reload {
		reload <- false

		// If no other options are specified, load the config file and run.
		c := NewConfig()
		err := c.LoadConfig(*fConfig)
		if err != nil {
			log.Fatal("E! " + err.Error())
		}

		if len(Inputs) == 0 {
			log.Fatalf("E! Error: no inputs found, did you provide a valid config file?")
		}

		fmt.Println("wooooooooooooooooooooo")

		if int64(c.Agent.Interval.Duration) <= 0 {
			log.Fatalf("E! Agent interval must be positive, found %s",
				c.Agent.Interval.Duration)
		}

		ag, err := NewAgent(c)
		if err != nil {
			log.Fatal("E! " + err.Error())
		}

		err = ag.Connect()
		if err != nil {
			log.Fatal("E! " + err.Error())
		}

		shutdown := make(chan struct{})
		signals := make(chan os.Signal)
		signal.Notify(signals, os.Interrupt, syscall.SIGHUP)
		go func() {
			select {
			case sig := <-signals:
				if sig == os.Interrupt {
					close(shutdown)
				}
				if sig == syscall.SIGHUP {
					log.Printf("I! Reloading Telegraf config\n")
					<-reload
					reload <- true
					close(shutdown)
				}
			case <-stop:
				close(shutdown)
			}
		}()

		log.Printf("I! Starting Telegraf %s\n", displayVersion())
		log.Printf("I! Loaded outputs: %s", strings.Join(c.OutputNames(), " "))
		log.Printf("I! Loaded inputs: %s", strings.Join(c.InputNames(), " "))
		log.Printf("I! Tags enabled: %s", c.ListTags())

		ag.Run(shutdown)
	}
}
