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

// Package ecscollector implements a Prometheus collector for Amazon ECS
// metrics available at the ECS metadata server.
package ecscollector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
)

// ECS cpu_stats are from upstream docker/moby. These values are in nanoseconds.
// https://github.com/moby/moby/blob/49f021ebf00a76d74f5ce158244083e2dfba26fb/api/types/stats.go#L18-L40
const (
	nanoSeconds = 1.0e9
	timeLayout = "2006-01-02T15:04:05.999999999Z"
	cpuIn1Vcpu = 1024
	bytesInMiB = 1024 * 1024
)


var (
	metadataDesc *prometheus.Desc
	svcCpuLimitDesc *prometheus.Desc
	svcMemLimitDesc *prometheus.Desc
	cpuTotalDesc *prometheus.Desc
	cpuUtilizedDesc *prometheus.Desc
	memoryUtilizedDesc *prometheus.Desc
	memUsageDesc *prometheus.Desc
	memLimitDesc *prometheus.Desc
	memCacheUsageDesc *prometheus.Desc
	networkRxBytesDesc *prometheus.Desc
	networkRxPacketsDesc *prometheus.Desc
	networkRxDroppedDesc *prometheus.Desc
	networkRxErrorsDesc *prometheus.Desc
	networkTxBytesDesc *prometheus.Desc
	networkTxPacketsDesc *prometheus.Desc
	networkTxDroppedDesc *prometheus.Desc
	networkTxErrorsDesc *prometheus.Desc

	labels []string
	svcLabels []string
	metadataLabels []string
	cpuLabels []string
	networkLabels []string

)

// NewCollector returns a new Collector that queries ECS metadata server
// for ECS task and container metrics.
func NewCollector(client *ecsmetadata.Client, customLabels map[string]string) prometheus.Collector {
	metadataLabels = []string{
		"cluster",
		"task_arn",
		"family",
		"revision",
		"desired_status",
		"known_status",
		"pull_started_at",
		"pull_stopped_at",
		"availability_zone",
		"launch_type",
		"task_id",
	}
	svcLabels = []string{
		"task_arn",
		"task_id",
	}
	labels = []string{
		"container",
		"task_id",
	}


	var customLabelKeys  []string
	var customLabelValues []string

	for key, value := range customLabels {
		customLabelKeys = append(customLabelKeys, key)
		customLabelValues = append(customLabelValues, value)
	}

	// Append all the custom labels to the default labels at the end.
	metadataLabels = append(metadataLabels, customLabelKeys...)
	labels = append(labels, customLabelKeys...)
	svcLabels = append(svcLabels, customLabelKeys...)
	networkLabels = append(
		labels,
		"device",
	)
	cpuLabels = append(
		labels,
		"cpu",
	)

	// Initialize all the metric descriptors.

	metadataDesc  = prometheus.NewDesc(
		"ecs_metadata_info",
		"ECS service metadata.",
		metadataLabels, nil)

	svcCpuLimitDesc = prometheus.NewDesc(
		"ecs_svc_cpu_limit",
		"Total CPU Limit. (1 unit = 1/1024th of a vCPU)",
		svcLabels, nil)

	svcMemLimitDesc = prometheus.NewDesc(
		"ecs_svc_memory_limit_bytes",
		"Total MEM Limit in bytes.",
		svcLabels, nil)

	cpuTotalDesc = prometheus.NewDesc(
		"ecs_cpu_seconds_total",
		"Total CPU usage in seconds.",
		cpuLabels, nil)

	cpuUtilizedDesc = prometheus.NewDesc(
		"ecs_cpu_utilized",
		"Total CPU usage. (1 unit = 1/1024th of a vCPU)",
		labels, nil)

	memoryUtilizedDesc = prometheus.NewDesc(
		"ecs_memory_utilized_mega_bytes",
		"Total memory utilized in MB.",
		labels, nil)

	memUsageDesc = prometheus.NewDesc(
		"ecs_memory_bytes",
		"Memory usage in bytes.",
		labels, nil)

	memLimitDesc = prometheus.NewDesc(
		"ecs_memory_limit_bytes",
		"Memory limit in bytes.",
		labels, nil)

	memCacheUsageDesc = prometheus.NewDesc(
		"ecs_memory_cache_usage",
		"Memory cache usage in bytes.",
		labels, nil)

	networkRxBytesDesc = prometheus.NewDesc(
		"ecs_network_receive_bytes_total",
		"Network recieved in bytes.",
		networkLabels, nil)

	networkRxPacketsDesc = prometheus.NewDesc(
		"ecs_network_receive_packets_total",
		"Network packets recieved.",
		networkLabels, nil)

	networkRxDroppedDesc = prometheus.NewDesc(
		"ecs_network_receive_dropped_total",
		"Network packets dropped in recieving.",
		networkLabels, nil)

	networkRxErrorsDesc = prometheus.NewDesc(
		"ecs_network_receive_errors_total",
		"Network errors in recieving.",
		networkLabels, nil)

	networkTxBytesDesc = prometheus.NewDesc(
		"ecs_network_transmit_bytes_total",
		"Network transmitted in bytes.",
		networkLabels, nil)

	networkTxPacketsDesc = prometheus.NewDesc(
		"ecs_network_transmit_packets_total",
		"Network packets transmitted.",
		networkLabels, nil)

	networkTxDroppedDesc = prometheus.NewDesc(
		"ecs_network_transmit_dropped_total",
		"Network packets dropped in transmit.",
		networkLabels, nil)

	networkTxErrorsDesc = prometheus.NewDesc(
		"ecs_network_transmit_errors_total",
		"Network errors in transmit.",
		networkLabels, nil)


	return &collector{client: client, customLabelValues: customLabelValues}
}

type collector struct {
	client *ecsmetadata.Client
	customLabelValues []string
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cpuTotalDesc
	ch <- cpuUtilizedDesc
	ch <- memoryUtilizedDesc
	ch <- memUsageDesc
	ch <- memLimitDesc
	ch <- memCacheUsageDesc
	ch <- networkRxBytesDesc
	ch <- networkRxPacketsDesc
	ch <- networkRxDroppedDesc
	ch <- networkRxErrorsDesc
	ch <- networkTxBytesDesc
	ch <- networkTxPacketsDesc
	ch <- networkTxDroppedDesc
	ch <- networkTxErrorsDesc
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	metadata, err := c.client.RetrieveTaskMetadata(ctx)
	if err != nil {
		log.Printf("Failed to retrieve metadata: %v", err)
		return
	}

	metadataLableVals :=[]string{
		metadata.Cluster,
		metadata.TaskARN,
		metadata.Family,
		metadata.Revision,
		metadata.DesiredStatus,
		metadata.KnownStatus,
		metadata.PullStartedAt,
		metadata.PullStoppedAt,
		metadata.AvailabilityZone,
		metadata.LaunchType,
		metadata.TaskID,
	}
	metadataLableVals = append(metadataLableVals, c.customLabelValues...)

	ch <- prometheus.MustNewConstMetric(
		metadataDesc,
		prometheus.GaugeValue,
		1.0,
		metadataLableVals...,
	)

	svcLableVals := []string{
		metadata.TaskARN,
		metadata.TaskID,
	}

	svcLableVals = append(svcLableVals, c.customLabelValues...)

	ch <- prometheus.MustNewConstMetric(
		svcCpuLimitDesc,
		prometheus.GaugeValue,
		float64(metadata.Limits.CPU) * 1024,
		svcLableVals...,
	)

	ch <- prometheus.MustNewConstMetric(
		svcMemLimitDesc,
		prometheus.GaugeValue,
		float64(metadata.Limits.Memory),
		svcLableVals...,
	)

	stats, err := c.client.RetrieveTaskStats(ctx)
	if err != nil {
		log.Printf("Failed to retrieve container stats: %v", err)
		return
	}

	for _, container := range metadata.Containers {
		s := stats[container.DockerID]
		if s == nil {
			log.Printf("Couldn't find container with ID %q in stats", container.DockerID)
			continue
		}

		labelVals := []string{
			container.Name,
			metadata.TaskID,

		}
		labelVals = append(labelVals, c.customLabelValues...)

		// Calculate CPU usage percentage
		cpu_delta := s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage
		// system_delta := s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage

		parsedReadTime, _ := time.Parse(timeLayout, s.Read)
		parsedPreReadTime, _ := time.Parse(timeLayout, s.PreRead)
		time_diff_since_last_read := parsedReadTime.Sub(parsedPreReadTime).Nanoseconds()

		cpu_usage_in_vcpu := (float64(cpu_delta) / float64(time_diff_since_last_read) ) * cpuIn1Vcpu
		ch <- prometheus.MustNewConstMetric(
			cpuUtilizedDesc,
			prometheus.GaugeValue,
			cpu_usage_in_vcpu,
			labelVals...,
		)

		for i, cpuUsage := range s.CPUStats.CPUUsage.PercpuUsage {
			cpu := fmt.Sprintf("%d", i)
			cpuUsageSeconds := float64(cpuUsage) / nanoSeconds
			ch <- prometheus.MustNewConstMetric(
				cpuTotalDesc,
				prometheus.CounterValue,
				cpuUsageSeconds,
				append(labelVals, cpu)...,
			)
		}

		cacheValue := 0.0
		if val, ok := s.MemoryStats.Stats["cache"]; ok {
			cacheValue = float64(val)

			memoryUtilizedInMegaBytes := (float64(s.MemoryStats.Usage) - cacheValue) / bytesInMiB
			ch <- prometheus.MustNewConstMetric(
				memoryUtilizedDesc,
				prometheus.GaugeValue,
				memoryUtilizedInMegaBytes,
				labelVals...,
			)
		}

		for desc, value := range map[*prometheus.Desc]float64{
			memUsageDesc:      float64(s.MemoryStats.Usage),
			memLimitDesc:      float64(s.MemoryStats.Limit),
			memCacheUsageDesc: cacheValue,
		} {
			ch <- prometheus.MustNewConstMetric(
				desc,
				prometheus.GaugeValue,
				value,
				labelVals...,
			)
		}

		// Network metrics per interface.
		for iface, netStats := range s.Networks {
			networkLabelVals := append(labelVals, iface)

			for desc, value := range map[*prometheus.Desc]float64{
				networkRxBytesDesc:   netStats.RxBytes,
				networkRxPacketsDesc: netStats.RxPackets,
				networkRxDroppedDesc: netStats.RxDropped,
				networkRxErrorsDesc:  netStats.RxErrors,
				networkTxBytesDesc:   netStats.TxBytes,
				networkTxPacketsDesc: netStats.TxPackets,
				networkTxDroppedDesc: netStats.TxDropped,
				networkTxErrorsDesc:  netStats.TxErrors,
			} {
				ch <- prometheus.MustNewConstMetric(
					desc,
					prometheus.CounterValue,
					value,
					networkLabelVals...,
				)
			}
		}
	}
}
