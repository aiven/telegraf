# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working
with code in this repository.

## Repository

This is Aiven's fork of
[influxdata/telegraf](https://github.com/influxdata/telegraf), a
plugin-driven agent for collecting, processing, aggregating, and writing
metrics. See `AIVEN_CHANGES.md` for the list of Aiven-specific patches on
top of upstream (e.g. `aiven-procstat`, ClickHouse replication-queue
metrics, the Aiven Postgresql output, Prometheus name-remapping) — check
that file before touching any plugin listed there, since changes must
stay compatible with the reasons the fork diverged.

## Common commands

```shell
make telegraf          # build the ./cmd/telegraf binary (CGO_ENABLED=0)
make test              # go test -short ./...  (unit tests, no integration)
make test-integration  # go test -run Integration ./...
make test-all          # fmtcheck + vet + full go test ./... (no -short)
make fmt               # gofmt -s -w across the repo
make vet               # go vet (excludes plugins/parsers/influx)
make lint-install      # install golangci-lint + markdownlint
make lint              # golangci-lint run + markdownlint .
make lint-branch       # golangci-lint run against just the current branch's diff
make tidy              # go mod tidy, fails if go.mod/go.sum changed
make docs              # regenerate plugin README sample-config sections from sample.conf
make config            # regenerate etc/telegraf.conf from current plugin set
```

Run a single package/test directly with plain `go test`, e.g.:

```shell
go test ./plugins/inputs/cpu/...
go test -run TestCPU ./plugins/inputs/cpu/...
```

Before opening a PR, upstream's own checklist is `make lint && make check
&& make check-deps && make test && make docs`.

PR titles must follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary)
— this is enforced by the `semantic.yml` check when a PR has only one
commit.

## Architecture

Telegraf is a plugin system built around four plugin kinds, each defined
by a small interface in the repo root:

- `input.go` — `Input` (polled via `Gather(Accumulator) error`) and
  `ServiceInput` (long-running, `Start`/`Stop`).
- `output.go` — `Output` (`Connect`/`Write`/`Close`), plus
  `AggregatingOutput`.
- `processor.go` — `Processor` (synchronous `Apply`) and
  `StreamingProcessor` (`Start`/`Add`/`Stop`, for async work).
- `aggregator.go` — `Aggregator` (`Add`/`Push`/`Reset`).

All four share `PluginDescriber` (`plugin.go`), which just requires
`SampleConfig() string`. Plugins optionally implement `Initializer`
(`Init() error`), `PluginWithID`, `StatefulPlugin` (state persisted
across restarts via `persister/`), or `ProbePlugin` (health probe for
`startup_error_behavior`).

**Plugin layout and registration**: every plugin lives under
`plugins/{inputs,outputs,processors,aggregators, parsers,serializers,
secretstores}/<name>/` with its own `README.md` and `sample.conf`. A
plugin registers itself in a package-level `init()` by calling e.g.
`inputs.Add("name", func() telegraf.Input { return &Plugin{...} })` (see
`plugins/inputs/registry.go`). The plugin's Go file also carries a
`//go:generate ../../../tools/readme_config_includer/generator`
directive that embeds `sample.conf` into the README at doc-build time
(`make docs`). To make a plugin buildable into the binary it must be
imported (for its `init()` side effect) from `plugins/<kind>/all/all.go`.

**Runtime flow**: `agent/agent.go` owns the main loop — it gathers from
inputs on `interval`, runs them through `models/running_*.go` wrappers
(metric filtering, precision, name/tag overrides), pushes through
processors and aggregators, and flushes to outputs on `flush_interval`.
`config/config.go` parses the TOML config file(s) into running plugin
instances; `models/` holds the `Running*` wrapper types that apply the
common config knobs (`namepass`, `tagexclude`, etc.) uniformly across
all plugin kinds.

**Custom/subset builds**: `tools/custom_builder` builds a Telegraf
binary containing only a chosen subset of plugins based on a config
file, rather than the ~300+ plugin full build. `tools/config_includer`
and `tools/readme_config_includer` are the codegen backing `make
docs`/`make config`.

**Metrics**: `metric.go` / `metric/` define the core `Metric` type
(name, tags, fields, timestamp) that flows between every plugin stage;
`models/` accumulators enforce the tag/field mutation rules for it.

## Style notes

- Format with `gofmt -s`; `goimports` for import ordering. Aim for lines
  under ~80 chars (soft limit, enforced by `lll` in golangci-lint but
  not strict).
- golangci-lint config is `.golangci.yml` (v2 syntax) — notable enabled
  linters beyond defaults: `gosec`, `revive`, `unparam`, `prealloc`,
  `testifylint`, `perfsprint`.
- AI-generated code contributions are not accepted upstream per
  `CONTRIBUTING.md` — be aware of this policy when preparing anything
  intended to go back upstream rather than staying as an Aiven-only
  patch.
