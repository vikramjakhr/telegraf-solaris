package main

type InfluxSerializer struct {
}

func (s *InfluxSerializer) Serialize(m Metric) ([]byte, error) {
	return m.Serialize(), nil
}
