package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func main() {
	nodeID := "metrics-node-1"
	// Proxy is running on 8080 now
	proxyURL := "http://localhost:8080/push"

	fmt.Printf("Starting metrics pusher for node %s to %s\n", nodeID, proxyURL)

	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		mfs, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			log.Printf("Error gathering metrics: %v", err)
			continue
		}

		metrics := make(map[string]float64)
		for _, mf := range mfs {
			name := mf.GetName()
			for _, m := range mf.GetMetric() {
				var val float64
				switch mf.GetType() {
				case dto.MetricType_GAUGE:
					val = m.GetGauge().GetValue()
				case dto.MetricType_COUNTER:
					val = m.GetCounter().GetValue()
				case dto.MetricType_UNTYPED:
					val = m.GetUntyped().GetValue()
				default:
					// Skip Summary and Histogram for this simple proxy explanation
					continue
				}
				metrics[name] = val
			}
		}

		data, err := json.Marshal(metrics)
		if err != nil {
			log.Printf("Error marshaling metrics: %v", err)
			continue
		}

		req, err := http.NewRequest("POST", proxyURL, bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Error creating request: %v", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("node", nodeID)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Failed to push metrics: %v", err)
			continue
		}
		resp.Body.Close()

		fmt.Printf("Pushed %d metrics\n", len(metrics))
	}
}
