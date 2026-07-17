# Reasons for this fork

## Input Plugins

### ClickHouse

* Add extra metrics to monitor the replication queue

### Elasticsearch

* add cross cluster replication metrics ( they dont work for elasticsearch but
  its a first step until we have an opensearch plugin )

### Aiven Procstat

* basically a clone of procstat containing incompatible changes that are
  likely not upstreamable
* needed a way to parse multiple unit files in invocation of `systemctl` for
  performance Reasons
* the way that telegraf provides ( globbing ) does not fit our systemd unit
  structure
* we need to check units inside of containers

### MySQL

* added aggregated IOPerf Stats ( probably upstreamable )

## Output Plugins

### Aiven Postgresql

* added postgresql output plugin from scratch to work with timescaledb (
  probably upstreamable, although influxdata is not keen on supporting
  timescaledb as it seems )
* predates the upstream postgresql plugin and was subsequently moved to the
  aiven prefix

## Serializers

### Prometheus and Prometheus Remote Write

* colons in metric and label names are always replaced with underscores,
  regardless of `prometheus_name_sanitization` mode ( legacy or utf8 );
  this is now our api and we have to keep it. Upstream's legacy mode
  already does this natively, so the patch only needs to cover the utf8
  mode ( see `plugins/serializers/prometheus/convert.go` )
