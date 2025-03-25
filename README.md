# Pingdom Metrics Exporter for Prometheus

Prometheus exporter for uptime metrics exposed by the Pingdom API.

This is a fork of <https://github.com/camptocamp/prometheus-pingdom-exporter> which seems to be unmaintained.
We will mostly only do dependency or maintenance updates here.

A Helm chart for this exporter can be found at: <https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-pingdom-exporter>

## Running

Make sure you expose the Pingdom API Token via the `PINGDOM_API_TOKEN`
environment variable:

```bash
# Expose the Pingdom API Token
export PINGDOM_API_TOKEN=<api-token>

# Run the binary with the default options
bin/pingdom-exporter
```

### Usage

```bash
bin/pingdom-exporter -h

Usage of bin/pingdom-exporter:
  -default-uptime-slo float
      default uptime SLO to be used when the check doesn't provide a uptime SLO tag (i.e. uptime_slo_999 to 99.9% uptime SLO) (default 99)
  -metrics-path string
      path under which to expose metrics (default "/metrics")
  -outage-check-period int
      time (in days) in which to retrieve outage data from the Pingdom API (default 7)
  -port int
      port to listen on (default 9158)
  -tags string
      tag list separated by commas
```

#### Supported Pingdom Tags

##### `uptime_slo_xxx`

This will instruct pingdom-exporter to use a custom SLO for the given check
instead of the default one of 99%. Some tag examples and their corresponding
SLOs:

- `uptime_slo_99` - 99%, same as default
- `uptime_slo_995` - 99.5%
- `uptime_slo_999` - 99.9%

##### `pingdom_exporter_ignored`

Checks with this tag won't have their metrics exported. Use this when you don't
want to disable some check just to have it excluded from the pingdom-exporter
metrics.

You can also set the `-tags` flag to only return metrics for checks that contain
the given tags.

### Docker Image

Docker image tags are published to [GHCR](https://github.com/kokuwaio/pingdom-exporter/pkgs/container/pingdom-exporter) as changes are incorporated into the main branch.

## Exported Metrics

| Metric Name                                         | Description                                                                                                   |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `pingdom_down_seconds`                              | Total down time within the outage check period, in seconds                                                    |
| `pingdom_outages_total`                             | Number of outages within the outage check period                                                              |
| `pingdom_slo_period_seconds`                        | Outage check period, in seconds (see `-outage-check-period` flag)                                             |
| `pingdom_tags_label`                                | Formats a tag based on a regular expression (`-parser-tags` and `-tag-format`) (1: formatted, 0: unformatted) |
| `pingdom_tags`                                      | The current tags of the check                                                                                 |
| `pingdom_up_seconds`                                | Total up time within the outage check period, in seconds                                                      |
| `pingdom_up`                                        | Was the last query on Pingdom API successful                                                                  |
| `pingdom_uptime_response_time_seconds`              | The response time of last test, in seconds                                                                    |
| `pingdom_uptime_slo_error_budget_available_seconds` | Number of seconds of downtime we can still have without breaking the uptime SLO                               |
| `pingdom_uptime_slo_error_budget_total_seconds`     | Maximum number of allowed downtime, in seconds, according to the uptime SLO                                   |
| `pingdom_uptime_status`                             | The current status of the check (1: up, 0: down)                                                              |

## Development

All relevant commands are exposed via Makefile targets:

```sh
# Build the binary
make

# Run the tests
make test

# Check linting rules
make lint

# Build Docker image
make image

# Push Docker images to registry
make publish
```
