# spectrum_virtualize_exporter

![Go](https://github.com/bluecmd/spectrum_virtualize_exporter/workflows/Go/badge.svg)

Prometheus exporter for IBM Spectrum Virtualize (e.g. Storwize V7000).

# Supported Metrics

 * `spectrum_power_watts`
 * `spectrum_temperature`
 * `spectrum_drive_status`
 * `spectrum_psu_status`
 * `spectrum_pool_capacity_bytes`
 * `spectrum_pool_free_bytes`
 * `spectrum_pool_status`
 * `spectrum_pool_used_bytes`
 * `spectrum_pool_volume_count`
 * `spectrum_pool_virtual_bytes`
 * `spectrum_pool_used_before_reduction_bytes`
 * `spectrum_pool_used_after_reduction_bytes`
 * `spectrum_pool_reclaimable_bytes`
 * `spectrum_node_compression_usage_ratio`
 * `spectrum_node_fc_bps`
 * `spectrum_node_fc_iops`
 * `spectrum_node_iscsi_bps`
 * `spectrum_node_iscsi_iops`
 * `spectrum_node_sas_bps`
 * `spectrum_node_sas_iops`
 * `spectrum_node_system_usage_ratio`
 * `spectrum_node_total_cache_usage_ratio`
 * `spectrum_node_write_cache_usage_ratio`
 * `spectrum_system_iops{layer,op}`
 * `spectrum_system_bytes_per_second{layer,op}`
 * `spectrum_system_latency_seconds{layer,op}`
 * `spectrum_fc_port_speed_bps`
 * `spectrum_fc_port_status`
 * `spectrum_ip_port_link_active`
 * `spectrum_ip_port_speed_bps`
 * `spectrum_ip_port_state`

## Usage

Example:

```
./spectrum_virtualize_exporter \
  -auth-file ~/spectrum-monitor.yaml \
  -extra-ca-cert ~/namecheap.ca.crt
```

Where `~/spectrum-monitor.yaml` contains pairs of Spectrum targets
and login information in the following format:

```
"https://my-v7000:7443":
  user: monitor
  password: passw0rd
"https://my-other-v7000:7443":
  user: monitor2
  password: passw0rd1
```

The flag `-extra-ca-cert` is useful as it appears that at least V7000 on the
8.2 version is unable to attach an intermediate CA.

`spectrum_system_*` come from `lssystemstats`: `layer` is `vdisk` (I/O from
hosts to volumes), `mdisk` (from the pools to their managed disks) or `drive`
(to the physical drives), `op` is `read` or `write`. They are the system's
latest 5 s sample, so throughput has a resolution of 1 MiB/s and latency
of 1 ms.

## SSH instead of REST

The REST server has been seen to stop answering after a while (V7000 at 8.4)
while the CLI over SSH keeps working. Use an `ssh://` target to run the same
queries as `lsfoo -delim ,` over SSH instead:

```
"ssh://my-v7000":
  user: monitor
  keyfile: /etc/spectrum/id_rsa   # and/or password
```

```
./spectrum_virtualize_exporter \
  -auth-file ~/spectrum-monitor.yaml \
  -known-hosts ~/spectrum-known-hosts
```

`-known-hosts` is an OpenSSH `known_hosts` file for the targets (for example
from `ssh-keyscan my-v7000`); `-insecure` skips host key checking instead.
The connection is kept open between scrapes and redialled when it breaks.
Only `ls*` list commands are ever sent. A Monitor role user is enough.


## Missing Metrics?

Please [file an issue](https://github.com/bluecmd/spectrum_virtualize_exporter/issues/new) describing what metrics you'd like to see.
Include as much details as possible please, e.g. how the perfect Prometheus metric would look for your use-case.
