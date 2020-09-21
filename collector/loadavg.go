package collector

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"io/ioutil"
	"strconv"
	"strings"
)

type ScrapeLoadavgInfo struct{}

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

const (
	linuxLoadavg = "linux_load"
)

// Name method
func (ScrapeLoadavgInfo) Name() string {
	return linuxLoadavg
}

// Version method
func (ScrapeLoadavgInfo) Version() float64 {
	return 1.0
}

// Help method
func (ScrapeLoadavgInfo) Help() string {
	return "linux load avg info"
}

// Scrape method
func (ScrapeLoadavgInfo) Scrape(ch chan<- prometheus.Metric, logger log.Logger) error {
	loadInfo, err := getLoad()

	if err != nil {
		return err
	}
	oneloadDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, linuxLoadavg, "avg1"),
		"1m load average",
		nil, nil,
	)
	fiveloadDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, linuxLoadavg, "avg5"),
		"5m load average",
		nil, nil,
	)
	fifloadDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, linuxLoadavg, "avg15"),
		"15m load average",
		nil, nil,
	)
	ch <- prometheus.MustNewConstMetric(oneloadDesc, prometheus.GaugeValue, loadInfo[0])
	ch <- prometheus.MustNewConstMetric(fiveloadDesc, prometheus.GaugeValue, loadInfo[1])
	ch <- prometheus.MustNewConstMetric(fifloadDesc, prometheus.GaugeValue, loadInfo[2])
	return nil
}

func getLoad() (loads []float64, err error) {
	data, err := ioutil.ReadFile(procFilePath("loadavg"))
	if err != nil {
		return nil, err
	}
	loads, err = parseLoad(string(data))
	if err != nil {
		return nil, err
	}

	return loads, nil
}

func parseLoad(data string) (loads []float64, err error) {
	loads = make([]float64, 3)
	parts := strings.Fields(data)
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected content in %s", procFilePath("loadavg"))
	}
	for i, load := range parts[0:3] {
		loads[i], err = strconv.ParseFloat(load, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse laod '%s':%w", load, err)
		}
	}
	return loads, nil
}
