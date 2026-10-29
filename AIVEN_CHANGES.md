# Reasons for this fork

## Input Plugins

### Aiven Procstat

* 2022-01-21
  * basically a clone of procstat containing incompatible changes that are
    likely not upstreamable
  * needed a way to parse multiple unit files in invocation of `systemctl` for
    performance Reasons
  * the way that telegraf provides ( globbing ) does not fit our systemd unit
    structure
  * we need to check units inside of containers
* 2025-12-06
  * collect per-process swap from memory maps, since `MemoryInfo()` reports no
    swap on Linux

### ClickHouse

* 2023-10-30
  * Add extra metrics to monitor the replication queue

### Elasticsearch

* 2022-01-21
  * add cross cluster replication metrics ( they dont work for elasticsearch but
    its a first step until we have an opensearch plugin )
* 2025-02-26
  * add remote-backed store stats for OpenSearch ( `remote_store_stats`,
    requires `local = true` )

### MySQL

* 2022-01-21
  * added aggregated IOPerf Stats ( probably upstreamable )

### Zookeeper

* 2026-06-25
  * parse `mntr` lines by splitting on the tab, so keys with special
    characters ( e.g. znode names with `%`, spaces or `:` ) no longer abort
    the whole scrape; unparseable lines are skipped with a warning

## Output Plugins

### Aiven Postgresql

* 2022-01-21
  * added postgresql output plugin from scratch to work with timescaledb (
    probably upstreamable, although influxdata is not keen on supporting
    timescaledb as it seems )
  * predates the upstream postgresql plugin and was subsequently moved to the
    aiven prefix

## Serializers

### Prometheus and Prometheus Remote Write

* 2022-01-21
  * colons in metric and label names are always replaced with underscores,
    regardless of `prometheus_name_sanitization` mode ( legacy or utf8 );
    this is now our api and we have to keep it. Upstream strips colons only
    from label names in legacy mode, so the patch covers metric names in
    legacy mode and both name types in utf8 mode ( see
    `plugins/serializers/prometheus/convert.go` )
  * the `prometheus_client` output uses the same sanitization, so it is
    covered as well
