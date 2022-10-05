# Reasons for this fork

## Input Plugins

### Elasticsearch

* add cross cluster replication metrics ( they dont work for elasticsearch but its a first step unil we have an opensearch plugin )

### Aiven Procstat

* basically a clone of procstat containing incompatible changes that are likely not upstreamable
* needed a way to parse multiple unit files in invocation of `systemctl` for performance Reasons
* the way that telegraf provides ( globbing ) does not fit our systemd unit structure
* we need to check units inside of containers

### MySQL

* added aggregated IOPerf Stats ( probably upstreamable )

## Output Plugins

### Postgresql

* added postgresql output plugin from scratch to work with timescaledb ( probably upstreamable, although influxdata is not keen on supporting timescaledb as it seems )

### Kafka

* added support to connect through socks proxy ( probably upstreamable )

### Websocket

* added support to connect through socks proxy ( probably upstreamable )

### Prometheus Client

* added incompatible metric name replacements ( not sure exactely why it was needed, but its now our api and we have to keep it )

## Serializers

### Prometheus and Prometheus Remote Write

* changes to make `Plugins.Prometheus Client` work for the same reasons as stated there
