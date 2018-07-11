package main

import (
	"time"
)

func InitAllInputs() {
	AddInput("cpu", func() Input {
		return &CPUStats{
			PerCPU:   true,
			TotalCPU: true,
			ps:       nil,
		}
	})

	AddInput("mem", func() Input {
		return &MemStats{ps: nil}
	})

	AddInput("disk", func() Input {
		return &DiskStats{ps: nil}
	})

	AddInput("http_response", func() Input {
		return &HTTPResponse{}
	})

	AddInput("apache", func() Input {
		return &Apache{}
	})

	AddInput("ping", func() Input {
		return &Ping{
			pingHost:     hostPinger,
			PingInterval: 1.0,
			Count:        1,
			Timeout:      1.0,
		}
	})

	AddInput("net_response", func() Input {
		return &NetResponse{}
	})

	AddInput("tomcat", func() Input {
		return &Tomcat{
			URL:      "http://127.0.0.1:8080/manager/status/all?XML=true",
			Username: "tomcat",
			Password: "s3cret",
			Timeout:  Duration{Duration: 5 * time.Second},
		}
	})

	AddInput("jboss", func() Input {
		return &JBoss{
			client: &RealHTTPClient{},
		}
	})

	AddInput("jboss4", func() Input {
		return &JBoss4{
			client: &RealHTTPClient{},
		}
	})

	AddInput("system", func() Input {
		return &SystemStats{}
	})

	AddInput("netstat_connections", func() Input {
		return &NetStatConnections{}
	})

	AddInput("processes", func() Input {
		return &Processes{}
	})

	AddInput("diskio", func() Input {
		return &DiskIOStats{}
	})

	AddInput("net", func() Input {
		return &NetIOStats{}
	})

	AddInput("swap", func() Input {
		return &SwapStats{}
	})

	AddInput("procstat", func() Input {
		return &Procstat{}
	})

	AddInput("ntpq", func() Input {
		n := &NTPQ{}
		n.runQ = n.runq
		return n
	})
}

func InitAllOutputs() {
	AddOutput("influxdb", func() Output { return newInflux() })
}
