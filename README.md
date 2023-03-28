# Awair Local Prometheus Exporter

It's a Prometheus Exporter that talks to a local Awair device over its local JSON API and exports its metrics.

## Installation

Install with the Go CLI or by downloading a precompiled binary from the Releases page.

```shell
$ go install github.com/ericvolp12/awair-local-prom-exporter@v0.0.4
```

Run the binary with default arguments or provide your own:

```shell
$ awair-local-prom-exporter --help
Usage of awair-local-prom-exporter:
  -awair_addresses string
        Comma-separated list of Awair air-data URLs (default "http://localhost/air-data/latest")
  -listen string
        Listen address (default "0.0.0.0")
  -poll_frequency string
        Time (seconds) to wait between polling devices (default "30s")
  -port uint
        Listen port number (default 2112)
```

### Configure Exporter with Systemd

Configure a Systemd Unit to run the exporter:

```shell
$ sudo cat << EOF > /etc/systemd/system/prometheus-awair-exporter.service
[Unit]
Description=Prometheus exporter for Awair Air Sensors
Documentation=https://github.com/ericvolp12/awair-local-prom-exporter

[Service]
Restart=always
User=prometheus
ExecStart=/<path_to_binary>/awair-local-prom-exporter --port 2155 --listen 0.0.0.0 --awair_addresses http://<local_awair_device_address>/air-data/latest --poll_frequency 30s

[Install]
WantedBy=multi-user.target
EOF
```

Enable the Systemd unit:

```shell
$ sudo systemctl enable prometheus-awair-exporter.service
```

### Configure Prometheus to Scrape Exporter

Assuming you're running your prometheus instance on the same host as the Systemd unit you can configure Prometheus to scrape as follows:

Open your `/etc/prometheus/prometheus.yml` config and add the following:

```yaml
scrape_configs:
    - job_name: awair
    scrape_interval: 30s
    static_configs:
    - targets:
        - 127.0.0.1:2155
```
