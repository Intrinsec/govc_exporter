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

### vCenter TLS

The vCenter certificate is not verified by default, a warning is logged at the
first login. Use `--collector.vc.ca-file` (`VC_CA_FILE`) to verify it against a
CA bundle, or `--no-collector.vc.insecure` to verify it against the system CAs.

### Usage

```shell
./govc_exporter --help
usage: govc_exporter --collector.vc.password=COLLECTOR.VC.PASSWORD --collector.vc.username=COLLECTOR.VC.USERNAME --collector.vc.url=COLLECTOR.VC.URL [<flags>]


Flags:
  -h, --[no-]help                Show context-sensitive help (also try --help-long and --help-man).
      --[no-]collector.intrinsec  
                                 Enable intrinsec specific features
      --collector.vc.password=COLLECTOR.VC.PASSWORD  
                                 vc api password ($VC_PASSWORD)
      --collector.vc.username=COLLECTOR.VC.USERNAME  
                                 vc api username ($VC_USERNAME)
      --collector.vc.url=COLLECTOR.VC.URL  
                                 vc api url ($VC_URL)
      --[no-]collector.vc.insecure  
                                 Skip vCenter TLS certificate verification. Use
                                 --no-collector.vc.insecure to verify it against the system CAs.
                                 ($VC_INSECURE)
      --collector.vc.ca-file=""  PEM file of the CAs used to verify the vCenter TLS certificate.
                                 Enables the verification. ($VC_CA_FILE)
      --[no-]collector.ds        Enable the ds collector (default: enabled).
      --[no-]collector.esx       Enable the esx collector (default: enabled).
      --[no-]collector.respool   Enable the respool collector (default: enabled).
      --[no-]collector.spod      Enable the spod collector (default: enabled).
      --[no-]collector.vm        Enable the vm collector (default: enabled).
      --web.listen-address=":9752"  
                                 Address on which to expose metrics and web interface.
      --web.telemetry-path="/metrics"  
                                 Path under which to expose metrics.
      --[no-]web.disable-exporter-metrics  
                                 Exclude metrics about the exporter itself (promhttp_*, process_*,
                                 go_*).
      --web.max-requests=40      Maximum number of parallel scrape requests. Use 0 to disable.
      --[no-]collector.disable-defaults  
                                 Set all collectors to disabled by default.
      --web.config=""            [EXPERIMENTAL] Path to config yaml file that can enable TLS or
                                 authentication.
      --log.level=info           Only log messages with the given severity or above. One of: [debug,
                                 info, warn, error]
      --log.format=logfmt        Output format of log messages. One of: [logfmt, json]
      --[no-]version             Show application version.
```
