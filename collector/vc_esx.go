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

type esxCollector struct {
	vcCollector
	uptimeSeconds  typedDesc
	rebootRequired typedDesc
	cpuCoresTotal  typedDesc
	availCPUMhz    typedDesc
	usedCPUMhz     typedDesc
	availMemBytes  typedDesc
	usedMemBytes   typedDesc
}

const (
	esxCollectorSubsystem = "esx"
)

func init() {
	registerCollector(esxCollectorSubsystem, defaultEnabled, NewEsxCollector)
}

// NewEsxCollector returns a new Collector exposing IpTables stats.
func NewEsxCollector(logger log.Logger) (Collector, error) {

	labels := []string{"vc", "dc", "cluster", "name", "version", "status"}

	res := esxCollector{
		uptimeSeconds: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, esxCollectorSubsystem, "uptime_seconds"),
			"esx host uptime", labels, nil), prometheus.CounterValue},
		rebootRequired: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, esxCollectorSubsystem, "reboot_required"),
			"esx reboot required", labels, nil), prometheus.CounterValue},
		cpuCoresTotal: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, esxCollectorSubsystem, "cpu_cores_total"),
			"esx number of  cores", labels, nil), prometheus.CounterValue},
		availCPUMhz: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, esxCollectorSubsystem, "avail_cpu_mhz"),
			"esx total cpu in mhz", labels, nil), prometheus.CounterValue},
		usedCPUMhz: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, esxCollectorSubsystem, "used_cpu_mhz"),
			"esx cpu usage in mhz", labels, nil), prometheus.GaugeValue},
		availMemBytes: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, esxCollectorSubsystem, "avail_mem_bytes"),
			"esx total memory in bytes", labels, nil), prometheus.GaugeValue},
		usedMemBytes: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, esxCollectorSubsystem, "used_mem_bytes"),
			"esx used memory in bytes", labels, nil), prometheus.GaugeValue},
	}
	res.logger = logger

	return &res, nil
}

func (c *esxCollector) Update(ch chan<- prometheus.Metric) (err error) {

	cache.Flush()

	err = c.apiConnect()
	if err != nil {
		level.Error(c.logger).Log("msg", "unable to connect", "err", err)
		return err
	}
	defer c.apiDisconnect()
	hss, err := c.apiRetrieve()
	if err != nil {
		level.Error(c.logger).Log("msg", "unable retrieve esx", "err", err)
		return err
	}

	vc := *vcURL

	level.Debug(c.logger).Log("msg", "esx host retrieved", "num", len(hss))

	for _, hs := range hss {

		summ := hs.Summary
		name := summ.Config.Name

		tmp := getParents(c.ctx, c.logger, c.client.Client, hs.ManagedEntity)
		version := summ.Config.Product.Version
		status := string(summ.OverallStatus)
		qs := summ.QuickStats
		mb := int64(1024 * 1024)
		labels := []string{vc, tmp.dc, tmp.cluster, name, version, status}

		ch <- c.uptimeSeconds.mustNewConstMetric(float64(qs.Uptime), labels...)

		var val float64
		if summ.RebootRequired {
			val = 1.0
		} else {
			val = 0
		}
		ch <- c.rebootRequired.mustNewConstMetric(val, labels...)

		ch <- c.cpuCoresTotal.mustNewConstMetric(float64(summ.Hardware.NumCpuCores), labels...)
		ch <- c.availCPUMhz.mustNewConstMetric(float64(int64(summ.Hardware.NumCpuCores)*int64(summ.Hardware.CpuMhz)), labels...)
		ch <- c.usedCPUMhz.mustNewConstMetric(float64(qs.OverallCpuUsage), labels...)
		ch <- c.availMemBytes.mustNewConstMetric(float64(summ.Hardware.MemorySize), labels...)
		ch <- c.usedMemBytes.mustNewConstMetric(float64(int64(qs.OverallMemoryUsage)*mb), labels...)
	}
	return nil
}

func (c *esxCollector) apiRetrieve() ([]mo.HostSystem, error) {
	var hss []mo.HostSystem

	m := view.NewManager(c.client.Client)
	v, err := m.CreateContainerView(
		c.ctx,
		c.client.ServiceContent.RootFolder,
		[]string{"HostSystem"},
		true,
	)
	if err != nil {
		return hss, err
	}
	defer c.destroyView(v)

	err = v.Retrieve(
		c.ctx,
		[]string{"HostSystem"},
		[]string{
			"parent",
			"summary",
		},
		&hss,
	)
	return hss, err
}
