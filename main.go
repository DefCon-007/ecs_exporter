// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus-community/ecs_exporter/ecscollector"
	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var addr string
var shouldIgnoreExporterMetrics bool

func main() {
	flag.StringVar(&addr, "addr", ":9779", "The address to listen on for HTTP requests.")
	flag.BoolVar(&shouldIgnoreExporterMetrics,
		"ignore-exporter-metrics",
		false,
		"Flag to stop the exporter should expose its own metrics. To enable it, just add `--ignore-exporter-metrics` to the command line.`")
	flag.Parse()

	client, err := ecsmetadata.NewClientFromEnvironment()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	if shouldIgnoreExporterMetrics {
		log.Printf("Exporter metrics will not be exposed.")

		// Create a new registry and exclude the default Go metrics
		registry := prometheus.NewRegistry()
		registry.MustRegister(ecscollector.NewCollector(client))
		http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	} else {
		log.Printf("Exporter metrics will be exposed.")
		prometheus.MustRegister(ecscollector.NewCollector(client))
		http.Handle("/metrics", promhttp.Handler())
	}

	http.Handle("/", http.RedirectHandler("/metrics", http.StatusMovedPermanently))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	log.Printf("Starting server at %q", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
