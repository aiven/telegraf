package zookeeper

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/testutil"
)

func TestInit(t *testing.T) {
	plugin := &Zookeeper{}

	require.NoError(t, plugin.Init())

	require.Len(t, plugin.Servers, 1)
	require.Equal(t, ":2181", plugin.Servers[0])

	require.Equal(t, config.Duration(5*time.Second), plugin.Timeout)

	require.Nil(t, plugin.tlsConfig)
}

func TestParseMntr(t *testing.T) {
	mntr := strings.Join([]string{
		"zk_version\t3.8.0",
		"zk_server_state\tleader",
		"zk_avg_latency\t1.5",
		"zk_packets_received\t42",
		// Per-namespace metric whose embedded znode name contains '%'.
		"zk_avg_action_update_service_users_privileges%key%4b39c570825d4773%done_write_per_namespace\t86.0",
		// znode names allow most characters, including spaces and ':'.
		"zk_avg_latency_for_my namespace:1\t7",
		// Unparseable lines (no tab / wrong prefix) must be skipped.
		"this is not a metric line",
		"not_zk_prefixed\t1",
	}, "\n") + "\n"

	plugin := &Zookeeper{ParseFloats: "float", Log: testutil.Logger{}}
	fields, state := plugin.parseMntr(strings.NewReader(mntr))

	require.Equal(t, "leader", state)
	require.Equal(t, int64(42), fields["packets_received"])
	require.InDelta(t, 1.5, fields["avg_latency"], 1e-9)
	require.InDelta(t, 86.0, fields["avg_action_update_service_users_privileges%key%4b39c570825d4773%done_write_per_namespace"], 1e-9)
	require.Equal(t, int64(7), fields["avg_latency_for_my namespace:1"])
	require.NotContains(t, fields, "server_state")
	require.NotContains(t, fields, "not_zk_prefixed")
}

func TestZookeeperGeneratesMetricsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	servicePort := "2181"
	container := testutil.Container{
		Image:        "zookeeper",
		ExposedPorts: []string{servicePort},
		Env: map[string]string{
			"ZOO_4LW_COMMANDS_WHITELIST": "mntr",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(servicePort),
			wait.ForLog("ZooKeeper audit is disabled."),
		),
	}
	err := container.Start()
	require.NoError(t, err, "failed to start container")
	defer container.Terminate()

	var testset = []struct {
		name      string
		zookeeper Zookeeper
	}{
		{
			name: "floats as strings",
			zookeeper: Zookeeper{
				Servers: []string{
					fmt.Sprintf("%s:%s", container.Address, container.Ports[servicePort]),
				},
			},
		},
		{
			name: "floats as floats",
			zookeeper: Zookeeper{
				Servers: []string{
					fmt.Sprintf("%s:%s", container.Address, container.Ports[servicePort]),
				},
				ParseFloats: "float",
			},
		},
	}
	for _, tt := range testset {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.zookeeper.Init())

			var acc testutil.Accumulator
			require.NoError(t, acc.GatherError(tt.zookeeper.Gather))

			intMetrics := []string{
				"max_latency",
				"min_latency",
				"packets_received",
				"packets_sent",
				"outstanding_requests",
				"znode_count",
				"watch_count",
				"ephemerals_count",
				"approximate_data_size",
				"open_file_descriptor_count",
				"max_file_descriptor_count",
			}

			for _, metric := range intMetrics {
				require.True(t, acc.HasInt64Field("zookeeper", metric), metric)
			}

			if tt.zookeeper.ParseFloats == "float" {
				require.True(t, acc.HasFloatField("zookeeper", "avg_latency"), "avg_latency not a float")
			} else {
				require.True(t, acc.HasStringField("zookeeper", "avg_latency"), "avg_latency not a string")
			}
		})
	}
}
