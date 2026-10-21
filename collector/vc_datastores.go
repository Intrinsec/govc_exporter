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

type datastoreCollector struct {
	logger     *slog.Logger
	capacity   typedDesc
	freeSpace  typedDesc
	accessible typedDesc
}

const (
	datastoreCollectorSubsystem = "ds"
)

func init() {
	registerCollector(datastoreCollectorSubsystem, defaultEnabled, NewDatastoreCollector)
}

// NewDatastoreCollector returns a new Collector exposing IpTables stats.
func NewDatastoreCollector(logger *slog.Logger) (Collector, error) {
	labels := []string{"vc", "dc", "name", "type", "cluster", "maintenance_mode"}

	res := datastoreCollector{
		capacity: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, datastoreCollectorSubsystem, "capacity_bytes"),
			"datastore capacity in bytes", labels, nil), prometheus.GaugeValue},
		freeSpace: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, datastoreCollectorSubsystem, "free_space_bytes"),
			"datastore freespace in bytes", labels, nil), prometheus.GaugeValue},
		accessible: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, datastoreCollectorSubsystem, "accessible"),
			"datastore is accessible", labels, nil), prometheus.GaugeValue},
	}
	res.logger = logger
	return &res, nil
}

func (c *datastoreCollector) Update(ch chan<- prometheus.Metric, s *scrape) error {
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

	c.logger.Debug("datastore retrieved", "num", len(items))

	for _, item := range items {
		summary := item.Summary
		name := summary.Name
		tmp := inv.parents(item.Parent)

		labels := []string{vc, tmp.dc, name, summary.Type, tmp.spod, summary.MaintenanceMode}
		ch <- c.capacity.mustNewConstMetric(float64(summary.Capacity), labels...)
		ch <- c.freeSpace.mustNewConstMetric(float64(summary.FreeSpace), labels...)
		ch <- c.accessible.mustNewConstMetric(b2f(summary.Accessible), labels...)

	}
	return nil
}

func (c *datastoreCollector) apiRetrieve(s *scrape) ([]mo.Datastore, error) {
	var items []mo.Datastore
	err := s.retrieveAll("Datastore", []string{
		"parent",
		"summary",
	}, &items)
	return items, err
}
