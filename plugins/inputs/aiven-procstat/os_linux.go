//go:build linux

package aiven_procstat

func addSwapToMemStats(proc Process, prefix string, fields map[string]interface{}) {
	memMaps, err := proc.MemoryMaps(true)
	if err == nil && memMaps != nil && len(*memMaps) > 0 {
		fields[prefix+"memory_swap"] = (*memMaps)[0].Swap
	}
}
