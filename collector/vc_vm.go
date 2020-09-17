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
	"encoding/json"
	"net/url"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type virtualMachineCollector struct {
	numCPU                       typedDesc
	numCoresPerSocket            typedDesc
	memoryBytes                  typedDesc
	overallCPUUsage              typedDesc
	overallCPUDemand             typedDesc
	guestMemoryUsage             typedDesc
	hostMemoryUsage              typedDesc
	distributedCPUEntitlement    typedDesc
	distributedMemoryEntitlement typedDesc
	staticCPUEntitlement         typedDesc
	staticMemoryEntitlement      typedDesc
	privateMemory                typedDesc
	sharedMemory                 typedDesc
	swappedMemory                typedDesc
	balloonedMemory              typedDesc
	consumedOverheadMemory       typedDesc
	ftLogBandwidth               typedDesc
	ftSecondaryLatency           typedDesc
	compressedMemory             typedDesc
	uptimeSeconds                typedDesc
	ssdSwappedMemory             typedDesc
	numSnapshot                  typedDesc
	diskCapacityBytes            typedDesc
	networkConnected             typedDesc
	ethernetDriverConnected      typedDesc
	logger                       log.Logger
	ctx                          context.Context
	client                       *govmomi.Client
}

const (
	virtualMachineCollectorSubsystem = "vm"
)

func init() {
	registerCollector(virtualMachineCollectorSubsystem, defaultEnabled, NewVirtualMachineCollector)
}

// NewVirtualMachineCollector returns a new Collector exposing IpTables stats.
func NewVirtualMachineCollector(logger log.Logger) (Collector, error) {

	labels := []string{
		"vc", "dc", "cluster", "esx", "pool",
		"name", "hostname", "guestfullname",
		"power_state", "overall_status",
		"tools_status", "tools_version",
	}
	if *useIsecSpecifics {
		labels = append(labels, "crit", "responsable", "service")
	}
	networkLabels := make([]string, len(labels))
	ethernetDevLabels := make([]string, len(labels))
	diskLabels := make([]string, len(labels))

	copy(networkLabels, labels)
	copy(ethernetDevLabels, labels)
	copy(diskLabels, labels)

	networkLabels = append(networkLabels, "network", "mac", "ip")
	ethernetDevLabels = append(ethernetDevLabels, "driver_model", "driver_mac", "driver_status")
	diskLabels = append(diskLabels, "vmdk")

	return &virtualMachineCollector{

		numCPU: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "cpu_number_total"),
			"vm number of cpu", labels, nil), prometheus.CounterValue},

		numCoresPerSocket: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "cores_number_per_socket_total"),
			"vm number of cores by socket", labels, nil), prometheus.CounterValue},

		memoryBytes: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "memory_bytes"),
			"vm memory in bytes", labels, nil), prometheus.GaugeValue},

		overallCPUUsage: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "overall_cpu_usage_mhz"),
			"vm overall CPU usage in MHz", labels, nil), prometheus.GaugeValue},

		overallCPUDemand: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "overall_cpu_demand_mhz"),
			"vm overall CPU demand in MHz", labels, nil), prometheus.GaugeValue},

		guestMemoryUsage: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "guest_memory_usage_bytes"),
			"vm guest memory usage in bytes", labels, nil), prometheus.GaugeValue},

		hostMemoryUsage: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "host_memory_usage_bytes"),
			"vm host memory usage in bytes", labels, nil), prometheus.GaugeValue},

		distributedCPUEntitlement: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "distributed_cpu_entitlement_mhz"),
			"vm distributed CPU entitlement in MHz", labels, nil), prometheus.GaugeValue},

		distributedMemoryEntitlement: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "distributed_memory_entitlement_bytes"),
			"vm distributed memory entitlement in bytes", labels, nil), prometheus.GaugeValue},

		staticCPUEntitlement: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "static_cpu_entitlement_mhz"),
			"vm static CPU entitlement in MHz", labels, nil), prometheus.GaugeValue},

		staticMemoryEntitlement: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "static_memory_entitlement_bytes"),
			"vm static memory entitlement in bytes", labels, nil), prometheus.GaugeValue},

		privateMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "private_memory_bytes"),
			"vm private memory in bytes", labels, nil), prometheus.GaugeValue},

		sharedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "shared_memory_bytes"),
			"vm shared memory in bytes", labels, nil), prometheus.GaugeValue},

		swappedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "swapped_memory_bytes"),
			"vm swapped memory in bytes", labels, nil), prometheus.GaugeValue},

		balloonedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "ballooned_memory_bytes"),
			"vm ballooned memory in bytes", labels, nil), prometheus.GaugeValue},

		consumedOverheadMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "consumed_overhead_memory_bytes"),
			"vm consumed overhead memory bytes", labels, nil), prometheus.GaugeValue},

		ftLogBandwidth: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "ft_log_bandwidth"),
			"vm ft log bandwidth", labels, nil), prometheus.GaugeValue},

		ftSecondaryLatency: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "ft_secondary_latency"),
			"vm ft secondary latency", labels, nil), prometheus.GaugeValue},

		compressedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "compressed_memory_bytes"),
			"vm compressed memory in bytes", labels, nil), prometheus.GaugeValue},

		uptimeSeconds: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "uptime_seconds"),
			"vm uptime in seconds", labels, nil), prometheus.CounterValue},

		ssdSwappedMemory: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "ssd_swapped_memory_bytes"),
			"vm ssd swapped memory in bytes", labels, nil), prometheus.GaugeValue},

		numSnapshot: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "snapshot_number_total"),
			"vm number of snapshot", labels, nil), prometheus.GaugeValue},

		diskCapacityBytes: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "disk_capacity_bytes"),
			"vm disk capacity bytes", diskLabels, nil), prometheus.GaugeValue},

		networkConnected: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "network_connected"),
			"vm network connected", networkLabels, nil), prometheus.GaugeValue},

		ethernetDriverConnected: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, virtualMachineCollectorSubsystem, "ethernet_driver_connected"),
			"vm ethernet driver connected", ethernetDevLabels, nil), prometheus.GaugeValue},

		logger: logger,
	}, nil
}

func (c *virtualMachineCollector) Update(ch chan<- prometheus.Metric) (err error) {

	cache.Flush()

	err = c.apiConnect()
	if err != nil {
		level.Error(c.logger).Log("msg", "unable to connect", "err", err)
		return err
	}
	defer c.apiDisconnect()
	items, err := c.apiRetrieve()
	if err != nil {
		level.Error(c.logger).Log("msg", "unable retrieve vm", "err", err)
		return err
	}

	vc := *vcURL

	level.Debug(c.logger).Log("msg", "virtual machine retrieved", "num", len(items))

	for _, item := range items {

		var esxName string
		var poolName string
		var parents Parents

		pool := getVMPool(c.ctx, c.logger, c.client.Client, item)
		if pool == nil {
			parents = getParents(c.ctx, c.logger, c.client.Client, item.ManagedEntity)
			poolName = "NONE"
		} else {
			parents = getParents(c.ctx, c.logger, c.client.Client, *pool)
			poolName = pool.Name
		}
		host := getVMHostSystem(c.ctx, c.logger, c.client.Client, item)
		if host == nil {
			esxName = "NONE"
		} else {
			esxName = host.Name
		}

		labelsValues := []string{
			vc,
			parents.dc,
			parents.cluster,
			esxName,
			poolName,
			item.Summary.Config.Name,
			item.Summary.Guest.HostName,
			item.Summary.Guest.GuestFullName,
			string(item.Runtime.PowerState),
			string(item.Summary.OverallStatus),
			string(item.Guest.ToolsStatus),
			item.Guest.ToolsVersion,
		}

		if *useIsecSpecifics {
			annotation := GetIsecAnnotation(item)
			labelsValues = append(
				labelsValues,
				annotation.Criticality,
				annotation.Responsable,
				annotation.Service,
			)
		}
		mb := int64(1024 * 1024)

		ch <- c.numCPU.mustNewConstMetric(float64(item.Config.Hardware.NumCPU), labelsValues...)
		ch <- c.numCoresPerSocket.mustNewConstMetric(float64(item.Config.Hardware.NumCoresPerSocket), labelsValues...)
		ch <- c.memoryBytes.mustNewConstMetric(float64(int64(item.Config.Hardware.MemoryMB)*mb), labelsValues...)
		ch <- c.overallCPUUsage.mustNewConstMetric(float64(item.Summary.QuickStats.OverallCpuUsage), labelsValues...)
		ch <- c.overallCPUDemand.mustNewConstMetric(float64(item.Summary.QuickStats.OverallCpuDemand), labelsValues...)
		ch <- c.guestMemoryUsage.mustNewConstMetric(float64(int64(item.Summary.QuickStats.GuestMemoryUsage)*mb), labelsValues...)
		ch <- c.hostMemoryUsage.mustNewConstMetric(float64(int64(item.Summary.QuickStats.HostMemoryUsage)*mb), labelsValues...)
		ch <- c.distributedCPUEntitlement.mustNewConstMetric(float64(item.Summary.QuickStats.DistributedCpuEntitlement), labelsValues...)
		ch <- c.distributedMemoryEntitlement.mustNewConstMetric(float64(int64(item.Summary.QuickStats.DistributedMemoryEntitlement)*mb), labelsValues...)
		ch <- c.staticCPUEntitlement.mustNewConstMetric(float64(item.Summary.QuickStats.StaticCpuEntitlement), labelsValues...)
		ch <- c.staticMemoryEntitlement.mustNewConstMetric(float64(int64(item.Summary.QuickStats.StaticMemoryEntitlement)*mb), labelsValues...)
		ch <- c.privateMemory.mustNewConstMetric(float64(int64(item.Summary.QuickStats.PrivateMemory)*mb), labelsValues...)
		ch <- c.sharedMemory.mustNewConstMetric(float64(int64(item.Summary.QuickStats.SharedMemory)*mb), labelsValues...)
		ch <- c.swappedMemory.mustNewConstMetric(float64(int64(item.Summary.QuickStats.SwappedMemory)*mb), labelsValues...)
		ch <- c.balloonedMemory.mustNewConstMetric(float64(int64(item.Summary.QuickStats.BalloonedMemory)*mb), labelsValues...)
		ch <- c.consumedOverheadMemory.mustNewConstMetric(float64(int64(item.Summary.QuickStats.ConsumedOverheadMemory)*mb), labelsValues...)
		ch <- c.ftLogBandwidth.mustNewConstMetric(float64(item.Summary.QuickStats.FtLogBandwidth), labelsValues...)
		ch <- c.ftSecondaryLatency.mustNewConstMetric(float64(item.Summary.QuickStats.FtSecondaryLatency), labelsValues...)
		ch <- c.compressedMemory.mustNewConstMetric(float64(int64(item.Summary.QuickStats.CompressedMemory)*mb), labelsValues...)
		ch <- c.uptimeSeconds.mustNewConstMetric(float64(item.Summary.QuickStats.UptimeSeconds), labelsValues...)
		ch <- c.ssdSwappedMemory.mustNewConstMetric(float64(int64(item.Summary.QuickStats.SsdSwappedMemory)*mb), labelsValues...)

		if item.Snapshot != nil {
			ch <- c.numSnapshot.mustNewConstMetric(float64(len(item.Snapshot.RootSnapshotList)), labelsValues...)
		} else {
			ch <- c.numSnapshot.mustNewConstMetric(0.0, labelsValues...)
		}

		edevices := GetEthernetDevices(item)
		for _, edev := range edevices {
			tmp := append(labelsValues, edev.typeName, edev.mac, edev.status)
			ch <- c.ethernetDriverConnected.mustNewConstMetric(b2f(edev.connected), tmp...)
		}
		networks := GetNetworks(item)
		for _, net := range networks {
			for _, ip := range net.ip {
				tmp := append(labelsValues, net.network, net.mac, ip)
				ch <- c.networkConnected.mustNewConstMetric(b2f(net.connected), tmp...)
			}
		}
		disks := GetDisks(item)
		for _, disk := range disks {
			tmp := append(labelsValues, disk.vmdk)
			ch <- c.diskCapacityBytes.mustNewConstMetric(float64(disk.capacity), tmp...)
		}
	}
	return nil
}

type IsecAnnotation struct {
	Criticality string `json:"crit"`
	Responsable string `json:"resp"`
	Service     string `json:"svc"`
}

func GetIsecAnnotation(vm mo.VirtualMachine) IsecAnnotation {
	tmp := IsecAnnotation{
		Service:     "not defined",
		Responsable: "not defined",
		Criticality: "not defined",
	}
	_ = json.Unmarshal([]byte(vm.Config.Annotation), &tmp)
	return tmp
}

type EthernetDevice struct {
	name      string
	typeName  string
	mac       string
	status    string
	connected bool
}

func GetEthernetDevices(vm mo.VirtualMachine) []EthernetDevice {
	devices := object.VirtualDeviceList(vm.Config.Hardware.Device)
	res := make([]EthernetDevice, 0, len(devices))
	for _, dev := range devices {

		switch md := dev.(type) {
		case types.BaseVirtualEthernetCard:
			status := "unknown"
			connected := false
			name := devices.Name(dev)
			typeName := strings.TrimPrefix(devices.TypeName(dev), "Virtual")
			d := dev.GetVirtualDevice()
			mac := md.GetVirtualEthernetCard().MacAddress
			if ca := d.Connectable; ca != nil {
				status = ca.Status
				connected = ca.Connected
			}
			res = append(res, EthernetDevice{
				name:      name,
				typeName:  typeName,
				mac:       mac,
				status:    status,
				connected: connected,
			})

		}
	}
	return res
}

type Network struct {
	network   string
	mac       string
	ip        []string
	connected bool
}

func GetNetworks(vm mo.VirtualMachine) []Network {
	res := make([]Network, 0, len(vm.Guest.Net))
	for _, net := range vm.Guest.Net {
		item := Network{
			network:   net.Network,
			mac:       net.MacAddress,
			ip:        net.IpAddress,
			connected: net.Connected,
		}
		res = append(res, item)
	}
	return res
}

type Disk struct {
	vmdk     string
	capacity int64
}

func GetDisks(vm mo.VirtualMachine) []Disk {
	disks := object.VirtualDeviceList(vm.Config.Hardware.Device).SelectByType((*types.VirtualDisk)(nil))
	res := make([]Disk, 0, len(disks))
	for _, d := range disks {
		disk := d.(*types.VirtualDisk)
		info := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
		res = append(res, Disk{
			vmdk:     info.FileName,
			capacity: disk.CapacityInBytes,
		})
	}
	return res
}

func (c *virtualMachineCollector) apiConnect() error {
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

func (c *virtualMachineCollector) apiDisconnect() {
	err := c.client.Logout(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
	c.ctx.Done()
}

func (c *virtualMachineCollector) destroyView(v *view.ContainerView) {
	err := v.Destroy(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "logout error", "err", err)
	}
}

func (c *virtualMachineCollector) apiRetrieve() ([]mo.VirtualMachine, error) {
	var items []mo.VirtualMachine

	m := view.NewManager(c.client.Client)
	v, err := m.CreateContainerView(
		c.ctx,
		c.client.ServiceContent.RootFolder,
		[]string{"VirtualMachine"},
		true,
	)
	if err != nil {
		return items, err
	}
	defer c.destroyView(v)

	err = v.Retrieve(
		c.ctx,
		[]string{"VirtualMachine"},
		[]string{
			"config",
			//"datatore",
			"guest",
			"guestHeartbeatStatus",
			"network",
			"parent",
			"resourceConfig",
			"resourcePool",
			"runtime",
			"snapshot",
			"summary",
		},
		&items,
	)
	return items, err
}
