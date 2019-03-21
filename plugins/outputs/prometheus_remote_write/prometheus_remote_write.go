package prometheus_remote_write

import (
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/internal/tls"
	"github.com/influxdata/telegraf/plugins/outputs"
	"github.com/influxdata/telegraf/plugins/outputs/prometheus_client"
)

func init() {
	outputs.Add("prometheus_remote_write", func() telegraf.Output {
		return &PrometheusRemoteWrite{}
	})
}

type PrometheusRemoteWrite struct {
	URL           string `toml:"url"`
	BasicUsername string `toml:"basic_username"`
	BasicPassword string `toml:"basic_password"`
	tls.ClientConfig

	client http.Client
}

var sampleConfig = `
  ## URL to send Prometheus remote write requests to.
  url = "http://localhost/push"

  ## Optional HTTP asic auth credentials.
  # basic_username = "username"
  # basic_password = "pa55w0rd"

  ## Optional TLS Config for use on HTTP connections.
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false
`

func (p *PrometheusRemoteWrite) Connect() error {
	tlsConfig, err := p.ClientConfig.TLSConfig()
	if err != nil {
		return err
	}

	p.client = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	return nil
}

func (p *PrometheusRemoteWrite) Close() error {
	return nil
}

func (p *PrometheusRemoteWrite) Description() string {
	return "Configuration for the Prometheus remote write client to spawn"
}

func (p *PrometheusRemoteWrite) SampleConfig() string {
	return sampleConfig
}

func (p *PrometheusRemoteWrite) Write(metrics []telegraf.Metric) error {
	var req prompb.WriteRequest

	for _, metric := range metrics {
		tags := metric.TagList()
		commonLabels := make([]prompb.Label, 0, len(tags))
		for _, tag := range tags {
			commonLabels = append(commonLabels, prompb.Label{
				Name:  prometheus_client.Sanitize(tag.Key),
				Value: tag.Value,
			})
		}

		for _, field := range metric.FieldList() {
			labels := make([]prompb.Label, len(commonLabels), len(commonLabels)+1)
			copy(labels, commonLabels)
			labels = append(labels, prompb.Label{
				Name:  "__name__",
				Value: metric.Name() + "_" + field.Key,
			})
			sort.Sort(byName(labels))

			// Ignore histograms and summaries.
			switch metric.Type() {
			case telegraf.Histogram, telegraf.Summary:
				continue
			}

			// Ignore string and bool fields.
			var value float64
			switch fv := field.Value.(type) {
			case int64:
				value = float64(fv)
			case uint64:
				value = float64(fv)
			case float64:
				value = fv
			default:
				continue
			}

			req.Timeseries = append(req.Timeseries, prompb.TimeSeries{
				Labels: labels,
				Samples: []prompb.Sample{{
					Timestamp: metric.Time().UnixNano() / int64(time.Millisecond),
					Value:     value,
				}},
			})
		}
	}

	buf, err := proto.Marshal(&req)
	if err != nil {
		return err
	}

	compressed := snappy.Encode(nil, buf)
	httpReq, err := http.NewRequest("POST", p.URL, bytes.NewReader(compressed))
	if err != nil {
		return err
	}
	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	httpReq.Header.Set("User-Agent", "Telegraf/"+internal.Version())
	if p.BasicUsername != "" || p.BasicPassword != "" {
		httpReq.SetBasicAuth(p.BasicUsername, p.BasicPassword)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("server returned HTTP status %s (%d)", resp.Status, resp.StatusCode)
	}
	return nil
}

type byName []prompb.Label

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }
