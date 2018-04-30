package main

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
}

func InitAllOutputs() {
	AddOutput("influxdb", func() Output { return newInflux() })
}
