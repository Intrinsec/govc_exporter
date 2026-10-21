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
	"testing"

	"github.com/vmware/govmomi/vim25/types"
)

func ref(kind, value string) *types.ManagedObjectReference {
	return &types.ManagedObjectReference{Type: kind, Value: value}
}

func TestInventoryParents(t *testing.T) {
	var (
		root    = ref("Folder", "group-d1")
		dc      = ref("Datacenter", "datacenter-1")
		hosts   = ref("Folder", "group-h1")
		cluster = ref("ClusterComputeResource", "domain-c1")
		pool    = ref("ResourcePool", "resgroup-1")
		ds      = ref("Folder", "group-s1")
		spod    = ref("StoragePod", "group-p1")
	)
	inv := inventory{
		*root:    {name: "Datacenters"},
		*dc:      {name: "DC0", parent: root},
		*hosts:   {name: "host", parent: dc},
		*cluster: {name: "C0", parent: hosts},
		*pool:    {name: "Resources", parent: cluster},
		*ds:      {name: "datastore", parent: dc},
		*spod:    {name: "POD0", parent: ds},
	}

	for _, tc := range []struct {
		name   string
		parent *types.ManagedObjectReference
		want   Parents
	}{
		{"no parent", nil, Parents{"NONE", "NONE", "NONE"}},
		{"in pool", pool, Parents{dc: "DC0", cluster: "C0", spod: "NONE"}},
		{"in storage pod", spod, Parents{dc: "DC0", cluster: "NONE", spod: "POD0"}},
		{"above datacenter", root, Parents{"NONE", "NONE", "NONE"}},
		{"unknown parent", ref("Folder", "group-x"), Parents{"ERROR", "ERROR", "ERROR"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := inv.parents(tc.parent); got != tc.want {
				t.Errorf("parents() = %+v, want %+v", got, tc.want)
			}
		})
	}

	if e, ok := inv.entry(pool); !ok || e.name != "Resources" {
		t.Errorf("entry(pool) = %+v, %v", e, ok)
	}
	if _, ok := inv.entry(nil); ok {
		t.Error("entry(nil) found")
	}
}
