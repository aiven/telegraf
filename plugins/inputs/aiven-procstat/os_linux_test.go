//go:build linux

package aiven_procstat

import (
	"testing"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf/testutil"
)

type swapTestProc struct {
	testProc
	memMaps *[]process.MemoryMapsStat
}

func (p *swapTestProc) MemoryMaps(bool) (*[]process.MemoryMapsStat, error) {
	return p.memMaps, nil
}

func TestGather_MemorySwapFromMemoryMaps(t *testing.T) {
	var acc testutil.Accumulator

	p := Procstat{
		Exe:             exe,
		createPIDFinder: pidFinder([]PID{pid}, nil),
		createProcess: func(PID) (Process, error) {
			return &swapTestProc{
				testProc: testProc{tags: make(map[string]string)},
				memMaps:  &[]process.MemoryMapsStat{{Swap: 4096}},
			}, nil
		},
	}
	require.NoError(t, acc.GatherError(p.Gather))

	m, ok := acc.Get("procstat")
	require.True(t, ok)
	require.Equal(t, uint64(4096), m.Fields["memory_swap"])
}

func TestGather_NoMemorySwapWithoutMemoryMaps(t *testing.T) {
	var acc testutil.Accumulator

	p := Procstat{
		Exe:             exe,
		createPIDFinder: pidFinder([]PID{pid}, nil),
		createProcess:   newTestProc,
	}
	require.NoError(t, acc.GatherError(p.Gather))

	require.True(t, acc.HasMeasurement("procstat"))
	require.False(t, acc.HasField("procstat", "memory_swap"))
}
