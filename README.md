# vmware VCenter prometheus metrics exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/intrinsec/govc_exporter)](https://goreportcard.com/report/github.com/intrinsec/govc_exporter)

Prometheus stand alone exporter for VCenter metrics. Metrics are fetched by govmomi api.

Works also with stand alone ESX without VCenter.

| Collectors          | Description |
| ------------------- | ----------- |
| `collector.ds`      | Datastore metrics collector |
| `collector.esx`     | ESX (HostSystem) metrics collector |
| `collector.respool` | ResourcePool metrics collector |
| `collector.spod`    | Datastore Cluster (StoragePod) metrics collector |
| `collector.vm`      | VirtualMachine metrics Collector |

## Building and running

### Build

```shell
make
```

### Running

```shell
export VC_URL=FIXME
export VC_USERNAME=FIXME
export VC_PASSWORD=FIXME
./govc_exporter <flags>
```

### Usage

```shell
./govc_exporter --help
usage: govc_exporter --collector.vc.password=COLLECTOR.VC.PASSWORD --collector.vc.username=COLLECTOR.VC.USERNAME --collector.vc.url=COLLECTOR.VC.URL [<flags>]

Flags:
  -h, --help                 Show context-sensitive help (also try --help-long and --help-man).
      --collector.vc.password=COLLECTOR.VC.PASSWORD  
                             vc api password
      --collector.vc.username=COLLECTOR.VC.USERNAME  
                             vc api username
      --collector.vc.url=COLLECTOR.VC.URL  
                             vc api username
      --collector.intrinsec  Enable intrinsec specific features
      --collector.ds         Enable the ds collector (default: enabled).
      --collector.esx        Enable the esx collector (default: enabled).
      --collector.respool    Enable the respool collector (default: enabled).
      --collector.spod       Enable the spod collector (default: enabled).
      --collector.vm         Enable the vm collector (default: enabled).
      --web.listen-address=":9752"  
                             Address on which to expose metrics and web interface.
      --web.telemetry-path="/metrics"  
                             Path under which to expose metrics.
      --web.disable-exporter-metrics  
                             Exclude metrics about the exporter itself (promhttp_*, process_*, go_*).
      --web.max-requests=40  Maximum number of parallel scrape requests. Use 0 to disable.
      --collector.disable-defaults  
                             Set all collectors to disabled by default.
      --web.config=""        [EXPERIMENTAL] Path to config yaml file that can enable TLS or authentication.
      --log.level=info       Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt    Output format of log messages. One of: [logfmt, json]
      --version              Show application version.
```
