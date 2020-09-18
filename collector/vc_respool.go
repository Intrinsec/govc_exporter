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

// +build !noesx

package collector

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
)

type resourcePoolCollector struct {
	vcCollector
	overallCPUUsage              typedDesc
	overallCPUDemand             typedDesc
	guestMemoryUsage             typedDesc
	hostMemoryUsage              typedDesc
	distributedCPUEntitlement    typedDesc
	distributedMemoryEntitlement typedDesc
	staticCPUEntitlement         typedDesc
	privateMemory                typedDesc
	sharedMemory                 typedDesc
	swappedMemory                typedDesc
	balloonedMemory              typedDesc
	overheadMemory               typedDesc
	consumedOverheadMemory       typedDesc
	compressedMemory             typedDesc
}

const (
	resourcePoolCollectorSubsystem = "respool"
)

func init() {
	registerCollector(resourcePoolCollectorSubsystem, defaultEnabled, NewResourcePoolCollector)
}

// NewResourcePoolCollector returns a new Collector exposing IpTables stats.
func NewResourcePoolCollector(logger log.Logger) (Collector, error) {
	labels := []string{"vc", "dc", "name"}

	res := resourcePoolCollector{
		overallCPUUsage: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "used_cpu_mhz"),
			"datastore overall CPU usage MHz", labels, nil), prometheus.GaugeValue},
		overallCPUDemand: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "demanded_cpu_mhz"),
			"datastore overall CPU demand MHz", labels, nil), prometheus.GaugeValue},
		guestMemoryUsage: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "guest_used_mem_bytes"),
			"datastore guest memory usage in bytes", labels, nil), prometheus.GaugeValue},
		hostMemoryUsage: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "host_used_mem_bytes"),
			"datastore host memory usage in bytes", labels, nil), prometheus.GaugeValue},
		distributedCPUEntitlement: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "distributed_cpu_entitlement_mhz"),
			"datastore distributed CPU entitlement", labels, nil), prometheus.GaugeValue},
		distributedMemoryEntitlement: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "distributed_mem_entitlement_bytes"),
			"datastore distributed memory entitlement", labels, nil), prometheus.GaugeValue},
		staticCPUEntitlement: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "static_cpu_entitlement_mhz"),
			"datastore static cpu entitlement", labels, nil), prometheus.GaugeValue},
		privateMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "private_mem_bytes"),
			"datastore private memory in bytes", labels, nil), prometheus.GaugeValue},
		sharedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "shared_mem_bytes"),
			"datastore shared memory in bytes", labels, nil), prometheus.GaugeValue},
		swappedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "swapped_mem_bytes"),
			"datastore swapped memory in bytes", labels, nil), prometheus.GaugeValue},
		balloonedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "ballooned_mem_bytes"),
			"datastore ballooned memory in bytes", labels, nil), prometheus.GaugeValue},
		overheadMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "overhead_mem_bytes"),
			"datastore overhead memory in bytes", labels, nil), prometheus.GaugeValue},
		consumedOverheadMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "consumed_overhead_mem_bytes"),
			"datastore consumed overhead memory in bytes", labels, nil), prometheus.GaugeValue},
		compressedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, resourcePoolCollectorSubsystem, "compressed_mem_bytes"),
			"datastore compressed memory in bytes", labels, nil), prometheus.GaugeValue},
	}
	res.logger = logger
	return &res, nil
}

func (c *resourcePoolCollector) Update(ch chan<- prometheus.Metric) (err error) {

	cache.Flush()

	err = c.apiConnect()
	if err != nil {
		level.Error(c.logger).Log("msg", "unable to connect", "err", err)
		return err
	}
	defer c.apiDisconnect()
	items, err := c.apiRetrieve()
	if err != nil {
		level.Error(c.logger).Log("msg", "unable retrieve esx", "err", err)
		return err
	}

	vc := *vcURL

	level.Debug(c.logger).Log("msg", "datastore retrieved", "num", len(items))

	for _, item := range items {
		summary := item.Summary.GetResourcePoolSummary()
		if summary == nil || summary.QuickStats == nil {
			continue
		}
		name := item.Summary.GetResourcePoolSummary().Name
		tmp := getParents(c.ctx, c.logger, c.client.Client, item.ManagedEntity)

		labels := []string{vc, tmp.dc, name}
		mb := int64(1024 * 1024)
		ch <- c.overallCPUUsage.mustNewConstMetric(float64(summary.QuickStats.OverallCpuUsage), labels...)
		ch <- c.overallCPUDemand.mustNewConstMetric(float64(summary.QuickStats.OverallCpuDemand), labels...)
		ch <- c.guestMemoryUsage.mustNewConstMetric(float64(summary.QuickStats.GuestMemoryUsage*mb), labels...)
		ch <- c.hostMemoryUsage.mustNewConstMetric(float64(summary.QuickStats.HostMemoryUsage*mb), labels...)
		ch <- c.distributedCPUEntitlement.mustNewConstMetric(float64(summary.QuickStats.DistributedCpuEntitlement), labels...)
		ch <- c.distributedMemoryEntitlement.mustNewConstMetric(float64(summary.QuickStats.DistributedMemoryEntitlement*mb), labels...)
		ch <- c.staticCPUEntitlement.mustNewConstMetric(float64(summary.QuickStats.StaticCpuEntitlement), labels...)
		ch <- c.privateMemory.mustNewConstMetric(float64(summary.QuickStats.PrivateMemory*mb), labels...)
		ch <- c.sharedMemory.mustNewConstMetric(float64(summary.QuickStats.SharedMemory*mb), labels...)
		ch <- c.swappedMemory.mustNewConstMetric(float64(summary.QuickStats.SwappedMemory*mb), labels...)
		ch <- c.balloonedMemory.mustNewConstMetric(float64(summary.QuickStats.BalloonedMemory*mb), labels...)
		ch <- c.overheadMemory.mustNewConstMetric(float64(summary.QuickStats.OverheadMemory*mb), labels...)
		ch <- c.consumedOverheadMemory.mustNewConstMetric(float64(summary.QuickStats.ConsumedOverheadMemory*mb), labels...)
		ch <- c.compressedMemory.mustNewConstMetric(float64(summary.QuickStats.CompressedMemory*mb), labels...)
	}
	return nil
}

func (c *resourcePoolCollector) apiRetrieve() ([]mo.ResourcePool, error) {
	var items []mo.ResourcePool

	m := view.NewManager(c.client.Client)
	v, err := m.CreateContainerView(
		c.ctx,
		c.client.ServiceContent.RootFolder,
		[]string{"ResourcePool"},
		true,
	)
	if err != nil {
		return items, err
	}
	defer c.destroyView(v)

	err = v.Retrieve(
		c.ctx,
		[]string{"ResourcePool"},
		[]string{
			"parent",
			"summary",
		},
		&items,
	)
	return items, err
}
