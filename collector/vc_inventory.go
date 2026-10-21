// Copyright 2026 Intrinsec
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

	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// inventoryKinds are the managed object types that can appear in the parent
// chain of a collected object, or be referenced by one (VM host and pool).
var inventoryKinds = []string{
	"Folder",
	"Datacenter",
	"ClusterComputeResource",
	"ComputeResource",
	"HostSystem",
	"ResourcePool",
	"VirtualApp",
	"StoragePod",
}

type inventoryEntry struct {
	name   string
	parent *types.ManagedObjectReference
}

// inventory maps a managed object to its name and parent. It is loaded with a
// single property collector call, so parents are resolved in memory instead
// of one API round-trip per object and per level.
type inventory map[types.ManagedObjectReference]inventoryEntry

func loadInventory(ctx context.Context, client *vim25.Client) (inventory, error) {
	m := view.NewManager(client)
	v, err := m.CreateContainerView(ctx, client.ServiceContent.RootFolder, inventoryKinds, true)
	if err != nil {
		return nil, err
	}
	// A view that fails to be destroyed only lives until the session ends.
	defer func() { _ = v.Destroy(ctx) }()

	var entities []mo.ManagedEntity
	if err := v.Retrieve(ctx, inventoryKinds, []string{"name", "parent"}, &entities); err != nil {
		return nil, err
	}
	inv := make(inventory, len(entities))
	for _, e := range entities {
		inv[e.Self] = inventoryEntry{name: e.Name, parent: e.Parent}
	}
	return inv, nil
}

// entry returns the entry of ref, ok is false when ref is nil or unknown.
func (inv inventory) entry(ref *types.ManagedObjectReference) (e inventoryEntry, ok bool) {
	if ref == nil {
		return e, false
	}
	e, ok = inv[*ref]
	return e, ok
}

type Parents struct {
	dc      string
	cluster string
	spod    string
}

// parents walks up from parent to the datacenter, collecting the storage pod,
// cluster and datacenter names on the way.
func (inv inventory) parents(parent *types.ManagedObjectReference) Parents {
	res := Parents{
		dc:      "NONE",
		cluster: "NONE",
		spod:    "NONE",
	}
	for cur := parent; cur != nil; {
		e, ok := inv[*cur]
		if !ok {
			return Parents{"ERROR", "ERROR", "ERROR"}
		}
		switch cur.Type {
		case "StoragePod":
			res.spod = e.name
		case "ClusterComputeResource":
			res.cluster = e.name
		case "Datacenter":
			res.dc = e.name
			return res
		}
		cur = e.parent
	}
	return res
}
