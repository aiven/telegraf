//go:build !linux

package aiven_procstat

func addSwapToMemStats(Process, string, map[string]interface{}) {}
