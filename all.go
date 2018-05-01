package main

import "time"

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
}

func InitAllOutputs() {
	AddOutput("influxdb", func() Output { return newInflux() })
}
