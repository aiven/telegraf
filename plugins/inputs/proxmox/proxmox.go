//go:generate ../../../tools/readme_config_includer/generator
package proxmox

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//go:embed sample.conf
var sampleConfig string

func (*Proxmox) SampleConfig() string {
	return sampleConfig
}

func (px *Proxmox) Init() error {
	// Set hostname as default node name for backwards compatibility
	if px.NodeName == "" {
		//nolint:errcheck // best attempt setting of NodeName
		hostname, _ := os.Hostname()
		px.NodeName = hostname
	}

	tlsCfg, err := px.ClientConfig.TLSConfig()
	if err != nil {
		return err
	}
	px.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
		Timeout: time.Duration(px.ResponseTimeout),
	}

	return nil
}

func (px *Proxmox) Gather(acc telegraf.Accumulator) error {
	if err := px.getNodeSearchDomain(); err != nil {
		return fmt.Errorf("getting search domain failed: %w", err)
	}

	gatherLxcData(px, acc)
	gatherQemuData(px, acc)

	return nil
}

func (px *Proxmox) getNodeSearchDomain() error {
	apiURL := "/nodes/" + px.NodeName + "/dns"
	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return fmt.Errorf("requesting data failed: %w", err)
	}

	var nodeDNS nodeDNS
	if err := json.Unmarshal(jsonData, &nodeDNS); err != nil {
		return fmt.Errorf("decoding message failed: %w", err)
	}
	px.nodeSearchDomain = nodeDNS.Data.Searchdomain

	return nil
}

func performRequest(px *Proxmox, apiURL, method string, data url.Values) ([]byte, error) {
	request, err := http.NewRequest(method, px.BaseURL+apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "PVEAPIToken="+px.APIToken)

	resp, err := px.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return responseBody, nil
}

func gatherLxcData(px *Proxmox, acc telegraf.Accumulator) {
	gatherVMData(px, acc, lxc)
}

func gatherQemuData(px *Proxmox, acc telegraf.Accumulator) {
	gatherVMData(px, acc, qemu)
}

func gatherVMData(px *Proxmox, acc telegraf.Accumulator, rt resourceType) {
	vmStats, err := getVMStats(px, rt)
	if err != nil {
		px.Log.Errorf("Error getting VM stats: %v", err)
		return
	}

	// For each VM add metrics to Accumulator
	for _, vmStat := range vmStats.Data {
		vmConfig, err := getVMConfig(px, vmStat.ID, rt)
		if err != nil {
			px.Log.Errorf("Error getting VM config: %v", err)
			return
		}

		if vmConfig.Data.Template == 1 {
			px.Log.Debugf("Ignoring template VM %s (%s)", vmStat.ID, vmStat.Name)
			continue
		}

		currentVMStatus, err := px.getCurrentVMStatus(rt, vmStat.ID)
		if err != nil {
			px.Log.Errorf("Error getting VM current VM status: %v", err)
			return
		}

		vmFQDN := vmConfig.Data.Hostname
		if vmFQDN == "" {
			vmFQDN = vmStat.Name
		}
		domain := vmConfig.Data.Searchdomain
		if domain == "" {
			domain = px.nodeSearchDomain
		}
		if domain != "" {
			vmFQDN += "." + domain
		}

		nodeFQDN := px.NodeName
		if px.nodeSearchDomain != "" {
			nodeFQDN += "." + domain
		}

		tags := map[string]string{
			"node_fqdn": nodeFQDN,
			"vm_name":   vmStat.Name,
			"vm_fqdn":   vmFQDN,
			"vm_type":   string(rt),
		}

		memMetrics := getByteMetrics(currentVMStatus.TotalMem, currentVMStatus.UsedMem)
		swapMetrics := getByteMetrics(currentVMStatus.TotalSwap, currentVMStatus.UsedSwap)
		diskMetrics := getByteMetrics(currentVMStatus.TotalDisk, currentVMStatus.UsedDisk)

		fields := map[string]interface{}{
			"status":               currentVMStatus.Status,
			"uptime":               jsonNumberToInt64(currentVMStatus.Uptime),
			"cpuload":              jsonNumberToFloat64(currentVMStatus.CPULoad),
			"mem_used":             memMetrics.used,
			"mem_total":            memMetrics.total,
			"mem_free":             memMetrics.free,
			"mem_used_percentage":  memMetrics.usedPercentage,
			"swap_used":            swapMetrics.used,
			"swap_total":           swapMetrics.total,
			"swap_free":            swapMetrics.free,
			"swap_used_percentage": swapMetrics.usedPercentage,
			"disk_used":            diskMetrics.used,
			"disk_total":           diskMetrics.total,
			"disk_free":            diskMetrics.free,
			"disk_used_percentage": diskMetrics.usedPercentage,
		}
		acc.AddFields("proxmox", fields, tags)
	}
}

func (px *Proxmox) getCurrentVMStatus(rt resourceType, id json.Number) (vmStat, error) {
	apiURL := "/nodes/" + px.NodeName + "/" + string(rt) + "/" + string(id) + "/status/current"
	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return vmStat{}, err
	}

	var currentVMStatus vmCurrentStats
	err = json.Unmarshal(jsonData, &currentVMStatus)
	if err != nil {
		return vmStat{}, err
	}

	return currentVMStatus.Data, nil
}

func getVMStats(px *Proxmox, rt resourceType) (vmStats, error) {
	apiURL := "/nodes/" + px.NodeName + "/" + string(rt)
	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return vmStats{}, err
	}

	var vmStatistics vmStats
	err = json.Unmarshal(jsonData, &vmStatistics)
	if err != nil {
		return vmStats{}, err
	}

	return vmStatistics, nil
}

func getVMConfig(px *Proxmox, vmID json.Number, rt resourceType) (vmConfig, error) {
	apiURL := "/nodes/" + px.NodeName + "/" + string(rt) + "/" + string(vmID) + "/config"
	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return vmConfig{}, err
	}

	var vmCfg vmConfig
	err = json.Unmarshal(jsonData, &vmCfg)
	if err != nil {
		return vmConfig{}, err
	}

	return vmCfg, nil
}

func getByteMetrics(total, used json.Number) metrics {
	int64Total := jsonNumberToInt64(total)
	int64Used := jsonNumberToInt64(used)
	int64Free := int64Total - int64Used
	usedPercentage := 0.0
	if int64Total != 0 {
		usedPercentage = float64(int64Used) * 100 / float64(int64Total)
	}

	return metrics{
		total:          int64Total,
		used:           int64Used,
		free:           int64Free,
		usedPercentage: usedPercentage,
	}
}

func jsonNumberToInt64(value json.Number) int64 {
	int64Value, err := value.Int64()
	if err != nil {
		return 0
	}

	return int64Value
}

func jsonNumberToFloat64(value json.Number) float64 {
	float64Value, err := value.Float64()
	if err != nil {
		return 0
	}

	return float64Value
}

func init() {
	inputs.Add("proxmox", func() telegraf.Input {
		return &Proxmox{
			requestFunction: performRequest,
		}
	})
}
