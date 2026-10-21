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
	"sync"

	"github.com/alecthomas/kingpin/v2"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
)

var useIsecSpecifics = kingpin.Flag("collector.intrinsec", "Enable intrinsec specific features").Default("false").Bool()

// scrape holds what the collectors share during one scrape: the vCenter
// session and the inventory used to resolve parents, loaded once on demand.
type scrape struct {
	ctx    context.Context
	client *govmomi.Client
	// inventory returns the inventory, loading it on first call.
	inventory func() (inventory, error)
}

func newScrape(ctx context.Context, client *govmomi.Client) *scrape {
	s := &scrape{ctx: ctx, client: client}
	s.inventory = sync.OnceValues(func() (inventory, error) {
		return loadInventory(ctx, client.Client)
	})
	return s
}

// retrieveAll fetches props of every managed object of the given kind.
func (s *scrape) retrieveAll(kind string, props []string, dst any) error {
	m := view.NewManager(s.client.Client)
	v, err := m.CreateContainerView(s.ctx, s.client.ServiceContent.RootFolder, []string{kind}, true)
	if err != nil {
		return err
	}
	// A view that fails to be destroyed only lives until the session ends.
	defer func() { _ = v.Destroy(s.ctx) }()

	return v.Retrieve(s.ctx, []string{kind}, props, dst)
}

func b2f(val bool) float64 {
	if val {
		return 1.0
	}
	return 0.0
}
