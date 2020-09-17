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
	"context"
	"net/url"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
)

type datastoreCollector struct {
	capacity   typedDesc
	freeSpace  typedDesc
	accessible typedDesc
	logger     log.Logger
	ctx        context.Context
	client     *govmomi.Client
}

const (
	datastoreCollectorSubsystem = "ds"
)

func init() {
	registerCollector(datastoreCollectorSubsystem, defaultEnabled, NewDatastoreCollector)
}

// NewDatastoreCollector returns a new Collector exposing IpTables stats.
func NewDatastoreCollector(logger log.Logger) (Collector, error) {
	labels := []string{"vc", "dc", "name", "type", "cluster", "maintenance_mode"}

	return &datastoreCollector{
		capacity: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, datastoreCollectorSubsystem, "capacity_bytes"),
			"datastore capacity in bytes", labels, nil), prometheus.GaugeValue},
		freeSpace: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, datastoreCollectorSubsystem, "free_space_bytes"),
			"datastore freespace in bytes", labels, nil), prometheus.GaugeValue},
		accessible: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, datastoreCollectorSubsystem, "accessible"),
			"datastore is accessible", labels, nil), prometheus.GaugeValue},

		logger: logger,
	}, nil
}

func (c *datastoreCollector) Update(ch chan<- prometheus.Metric) (err error) {

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
		summary := item.Summary
		name := summary.Name
		tmp := getParents(c.ctx, c.logger, c.client.Client, item.ManagedEntity)

		labels := []string{vc, tmp.dc, name, summary.Type, tmp.spod, summary.MaintenanceMode}
		ch <- c.capacity.mustNewConstMetric(float64(summary.Capacity), labels...)
		ch <- c.freeSpace.mustNewConstMetric(float64(summary.FreeSpace), labels...)
		ch <- c.accessible.mustNewConstMetric(b2f(summary.Accessible), labels...)

	}
	return nil
}

func (c *datastoreCollector) apiConnect() error {
	esxURL := *vcURL
	level.Debug(c.logger).Log("msg", "connecting to esx", "url", esxURL)
	u, err := soap.ParseURL(esxURL)
	if err != nil {
		level.Error(c.logger).Log("msg", "unable to parse url", "url", esxURL, "err", err)
		return err
	}
	u.User = url.UserPassword(*vcUsername, *vcPassword)
	c.ctx = context.Background()
	c.client, err = govmomi.NewClient(c.ctx, u, true)
	return err
}

func (c *datastoreCollector) apiDisconnect() {
	err := c.client.Logout(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
	c.ctx.Done()
}

func (c *datastoreCollector) destroyView(v *view.ContainerView) {
	err := v.Destroy(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
}

func (c *datastoreCollector) apiRetrieve() ([]mo.Datastore, error) {
	var items []mo.Datastore

	m := view.NewManager(c.client.Client)
	v, err := m.CreateContainerView(
		c.ctx,
		c.client.ServiceContent.RootFolder,
		[]string{"Datastore"},
		true,
	)
	if err != nil {
		return items, err
	}
	defer c.destroyView(v)

	err = v.Retrieve(
		c.ctx,
		[]string{"Datastore"},
		[]string{
			"parent",
			"summary",
		},
		&items,
	)
	return items, err
}
