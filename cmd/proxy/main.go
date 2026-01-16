package main

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net/http"
	"sync"
)

// MetricPayload represents the JSON body sent by the metrics agent.
type MetricPayload map[string]float64

// NodeMetrics stores the metrics for a specific node.
type NodeMetrics struct {
	Metrics MetricPayload
	mu      sync.RWMutex
}

var (
	store   = make(map[string]*NodeMetrics)
	storeMu sync.RWMutex
)

func main() {
	// Handler to receive metrics from cmd/metrics
	http.HandleFunc("/push", handlePush)

	// Handler for Prometheus to query
	http.HandleFunc("/metrics", handleMetrics)

	port := ":8080"
	fmt.Printf("Proxy Server starting on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func handlePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Identify the source node
	node := r.Header.Get("node")
	if node == "" {
		http.Error(w, "missing node header", http.StatusBadRequest)
		return
	}

	var payload MetricPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storeMu.Lock()
	nm, ok := store[node]
	if !ok {
		nm = &NodeMetrics{Metrics: make(MetricPayload)}
		store[node] = nm
	}
	storeMu.Unlock()

	nm.mu.Lock()
	maps.Copy(nm.Metrics, payload)
	nm.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	storeMu.RLock()
	defer storeMu.RUnlock()

	for node, nm := range store {
		nm.mu.RLock()
		for k, v := range nm.Metrics {
			// Format: metric_name{node="node_id"} value
			fmt.Fprintf(w, "%s{node=\"%s\"} %f\n", k, node, v)
		}
		nm.mu.RUnlock()
	}
}
