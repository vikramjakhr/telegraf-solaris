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
}

func InitAllOutputs() {
	AddOutput("influxdb", func() Output { return newInflux() })
}
