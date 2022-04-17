package goservice

import (
	"encoding/json"

	"github.com/zserge/metric"
)

type CountSample struct {
	Type  string  `json:"type"`
	Count float64 `json:"count"`
}
type Count struct {
	Interval int           `json:"interval"`
	Samples  []CountSample `json:"samples"`
}

func MetricsGetValueCounter(m metric.Metric) float64 {
	var data Count
	json.Unmarshal([]byte(m.String()), &data)
	total := 0
	for i := 0; i < len(data.Samples); i++ {
		total += int(data.Samples[i].Count)
	}
	return float64(total)
}

const (
	MCountCall string = "count_call"
)
const (
	MCountCallTime string = "1h1h"
)

func initMetrics() {
}
