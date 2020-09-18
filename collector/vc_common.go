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

package collector

import (
	"context"
	"net/url"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	vcPassword       = kingpin.Flag("collector.vc.password", "vc api password").Envar("VC_PASSWORD").Required().String()
	vcUsername       = kingpin.Flag("collector.vc.username", "vc api username").Envar("VC_USERNAME").Required().String()
	vcURL            = kingpin.Flag("collector.vc.url", "vc api username").Envar("VC_URL").Required().String()
	useIsecSpecifics = kingpin.Flag("collector.intrinsec", "Enable intrinsec specific features").Default("false").Bool()
	cache            = NewParentsCache()
)

type Parents struct {
	dc      string
	cluster string
	spod    string
}

type ParentsCache struct {
	cache map[types.ManagedObjectReference]Parents
	mux   sync.Mutex
}

func NewParentsCache() *ParentsCache {
	return &ParentsCache{
		cache: make(map[types.ManagedObjectReference]Parents),
	}
}

func (c *ParentsCache) Add(mor types.ManagedObjectReference, val Parents) {
	c.mux.Lock()
	c.cache[mor] = val
	c.mux.Unlock()
}

func (c *ParentsCache) Get(mor types.ManagedObjectReference) (Parents, bool) {
	c.mux.Lock()
	val, ok := c.cache[mor]
	c.mux.Unlock()
	return val, ok
}

func (c *ParentsCache) Flush() {
	c.mux.Lock()
	c.cache = make(map[types.ManagedObjectReference]Parents)
	c.mux.Unlock()
}

func getParents(ctx context.Context, logger log.Logger, client *vim25.Client, me mo.ManagedEntity) Parents {
	var entity mo.ManagedEntity
	var cur *types.ManagedObjectReference
	res := Parents{
		dc:      "NONE",
		cluster: "NONE",
		spod:    "NONE",
	}

	if me.Parent == nil {
		return res
	}
	cached, ok := cache.Get(*me.Parent)
	if ok {
		return cached
	}

	pc := property.DefaultCollector(client)

	cur = me.Parent
	for {
		err := pc.RetrieveOne(ctx, *cur, []string{"name", "parent"}, &entity)
		if err != nil {
			return Parents{"ERROR", "ERROR", "ERROR"}
		}
		if cur.Type == "StoragePod" {
			res.spod = entity.Name
		}
		if cur.Type == "ClusterComputeResource" {
			res.cluster = entity.Name
		}
		if cur.Type == "Datacenter" {
			res.dc = entity.Name
			break
		}
		if entity.Parent == nil {
			break
		}
		cur = entity.Parent
	}
	cache.Add(*me.Parent, res)
	return res
}

func getVMPool(ctx context.Context, logger log.Logger, client *vim25.Client, me mo.VirtualMachine) *mo.ManagedEntity {
	if me.ResourcePool == nil {
		return nil
	}

	var entity mo.ManagedEntity
	pc := property.DefaultCollector(client)
	err := pc.RetrieveOne(ctx, *me.ResourcePool, []string{"name", "parent"}, &entity)
	if err != nil {
		return nil
	}
	return &entity
}

func getVMHostSystem(ctx context.Context, logger log.Logger, client *vim25.Client, me mo.VirtualMachine) *mo.ManagedEntity {
	if me.Summary.Runtime.Host == nil {
		return nil
	}

	var entity mo.ManagedEntity
	pc := property.DefaultCollector(client)
	err := pc.RetrieveOne(ctx, *me.Summary.Runtime.Host, []string{"name", "parent"}, &entity)
	if err != nil {
		return nil
	}
	return &entity
}

func b2f(val bool) float64 {
	if val {
		return 1.0
	}
	return 0.0
}

type vcCollector struct {
	logger log.Logger
	ctx    context.Context
	client *govmomi.Client
}

func (c *vcCollector) apiConnect() error {
	esxURL := *vcURL
	level.Debug(c.logger).Log("msg", "connecting to", "url", esxURL)
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

func (c *vcCollector) apiDisconnect() {
	err := c.client.Logout(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
	c.ctx.Done()
}

func (c *vcCollector) destroyView(v *view.ContainerView) {
	err := v.Destroy(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
}
