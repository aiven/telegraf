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

### Elasticsearch

* 2022-01-21
  * add cross cluster replication metrics ( they dont work for elasticsearch but
    its a first step until we have an opensearch plugin )

### MySQL

* 2022-01-21
  * added aggregated IOPerf Stats ( probably upstreamable )

## Output Plugins

### Aiven Postgresql

* 2022-01-21
  * added postgresql output plugin from scratch to work with timescaledb (
    probably upstreamable, although influxdata is not keen on supporting
    timescaledb as it seems )
  * predates the upstream postgresql plugin and was subsequently moved to the
    aiven prefix
