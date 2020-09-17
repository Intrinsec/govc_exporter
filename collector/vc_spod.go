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

type storagePodCollector struct {
	capacity  typedDesc
	freeSpace typedDesc
	logger    log.Logger
	ctx       context.Context
	client    *govmomi.Client
}

const (
	storagePodCollectorSubsystem = "spod"
)

func init() {
	registerCollector(storagePodCollectorSubsystem, defaultEnabled, NewStoragePodCollector)
}

// NewStoragePodCollector returns a new Collector exposing IpTables stats.
func NewStoragePodCollector(logger log.Logger) (Collector, error) {
	labels := []string{"vc", "dc", "name"}

	return &storagePodCollector{
		capacity: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, storagePodCollectorSubsystem, "capacity_bytes"),
			"storagePod capacity in bytes", labels, nil), prometheus.GaugeValue},
		freeSpace: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, storagePodCollectorSubsystem, "free_space_bytes"),
			"storagePod freespace in bytes", labels, nil), prometheus.GaugeValue},

		logger: logger,
	}, nil
}

func (c *storagePodCollector) Update(ch chan<- prometheus.Metric) (err error) {

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

	level.Debug(c.logger).Log("msg", "storagePod retrieved", "num", len(items))

	for _, item := range items {
		summary := item.Summary
		name := summary.Name
		tmp := getParents(c.ctx, c.logger, c.client.Client, item.ManagedEntity)

		labels := []string{vc, tmp.dc, name}
		ch <- c.capacity.mustNewConstMetric(float64(summary.Capacity), labels...)
		ch <- c.freeSpace.mustNewConstMetric(float64(summary.FreeSpace), labels...)

	}
	return nil
}

func (c *storagePodCollector) apiConnect() error {
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

func (c *storagePodCollector) apiDisconnect() {
	err := c.client.Logout(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
	c.ctx.Done()
}

func (c *storagePodCollector) destroyView(v *view.ContainerView) {
	err := v.Destroy(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
}

func (c *storagePodCollector) apiRetrieve() ([]mo.StoragePod, error) {
	var items []mo.StoragePod

	m := view.NewManager(c.client.Client)
	v, err := m.CreateContainerView(
		c.ctx,
		c.client.ServiceContent.RootFolder,
		[]string{"StoragePod"},
		true,
	)
	if err != nil {
		return items, err
	}
	defer c.destroyView(v)

	err = v.Retrieve(
		c.ctx,
		[]string{"StoragePod"},
		[]string{
			"parent",
			"summary",
		},
		&items,
	)
	return items, err
}
