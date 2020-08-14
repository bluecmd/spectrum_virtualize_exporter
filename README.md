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


## Missing Metrics?

Please [file an issue](https://github.com/bluecmd/spectrum_virtualize_exporter/issues/new) describing what metrics you'd like to see.
Include as much details as possible please, e.g. how the perfect Prometheus metric would look for your use-case.
