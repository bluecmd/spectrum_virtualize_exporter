# spectrum_virtualize_exporter

Prometheus exporter for IBM Spectrum Virtualize (e.g. Storwize V7000).

# Supported Metrics

 * `spectrum_power_watts`

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
