// Copyright 2020 Intrinsec
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

//go:build !noesx

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vmware/govmomi/vim25/mo"
)

type storagePodCollector struct {
	logger    *slog.Logger
	capacity  typedDesc
	freeSpace typedDesc
}

const (
	storagePodCollectorSubsystem = "spod"
)

func init() {
	registerCollector(storagePodCollectorSubsystem, defaultEnabled, NewStoragePodCollector)
}

// NewStoragePodCollector returns a new Collector exposing IpTables stats.
func NewStoragePodCollector(logger *slog.Logger) (Collector, error) {
	labels := []string{"vc", "dc", "name"}

	res := storagePodCollector{
		capacity: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, storagePodCollectorSubsystem, "capacity_bytes"),
			"storagePod capacity in bytes", labels, nil), prometheus.GaugeValue},
		freeSpace: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, storagePodCollectorSubsystem, "free_space_bytes"),
			"storagePod freespace in bytes", labels, nil), prometheus.GaugeValue},
	}
	res.logger = logger
	return &res, nil
}

func (c *storagePodCollector) Update(ch chan<- prometheus.Metric, s *scrape) error {
	items, err := c.apiRetrieve(s)
	if err != nil {
		c.logger.Error("unable retrieve esx", "err", err)
		return err
	}

	inv, err := s.inventory()
	if err != nil {
		c.logger.Error("unable to retrieve inventory", "err", err)
		return err
	}

	vc := *vcURL

	c.logger.Debug("storagePod retrieved", "num", len(items))

	for _, item := range items {
		summary := item.Summary
		name := summary.Name
		tmp := inv.parents(item.Parent)

		labels := []string{vc, tmp.dc, name}
		ch <- c.capacity.mustNewConstMetric(float64(summary.Capacity), labels...)
		ch <- c.freeSpace.mustNewConstMetric(float64(summary.FreeSpace), labels...)

	}
	return nil
}

func (c *storagePodCollector) apiRetrieve(s *scrape) ([]mo.StoragePod, error) {
	var items []mo.StoragePod
	err := s.retrieveAll("StoragePod", []string{
		"parent",
		"summary",
	}, &items)
	return items, err
}
