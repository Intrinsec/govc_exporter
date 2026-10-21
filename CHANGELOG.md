## Unreleased
* [CHANGE] Go 1.27.3 pinned in `go.mod`, `Dockerfile` and CI; all dependencies updated (govmomi v0.56.0, client_golang v1.25.0, common v0.72.1).
* [CHANGE] `--web.config` is now handled by [exporter-toolkit](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md): the `tls_config` key is renamed `tls_server_config`. An old file makes the exporter fail at startup.
* [CHANGE] Logging uses `log/slog` (`--log.level`/`--log.format` unchanged, output now includes `source=`).
* [ENHANCEMENT] Multi-stage `Dockerfile`, no local build needed (#5); multi-arch image built with buildx.
* [ENHANCEMENT] GitHub Actions CI workflow: tidy/vendor check, vet, race tests, golangci-lint v2.
* [ENHANCEMENT] Parents (dc, cluster, storage pod, VM host and pool) are resolved from an inventory loaded in one call per scrape instead of one call per object and level: about 13x faster and 60x fewer vCenter calls on a 1000 VM simulated inventory. Series are unchanged.
* [ENHANCEMENT] One vCenter session is shared by all scrapes and renewed only when it expires, instead of a login/logout per collector and per scrape.
* [ENHANCEMENT] `--collector.vc.ca-file` and `--[no-]collector.vc.insecure` to verify the vCenter TLS certificate; it is still not verified by default, with a warning.
* [BUGFIX] Concurrent scrapes no longer share per collector connection state.
* [BUGFIX] `govc_vm_cores_number_per_socket_total` handles the now optional `NumCoresPerSocket` (0 when unset).

## 0.1.0 / 2019-05-23
* [CLEANUP] Made stand alone from refused PR
