package collector

import (
	"bufio"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	reParens   = regexp.MustCompile(`\((.*)\)`)
	memInfo    = map[string]float64{}
	memoryinfo = "linux_memory_info"
)

type ScrapeMemoryInfo struct{}

// Name method
func (ScrapeMemoryInfo) Name() string {
	return memoryinfo
}

// Help method
func (ScrapeMemoryInfo) Help() string {
	return "Scrape memory info"
}

// Version method
func (ScrapeMemoryInfo) Version() float64 {
	return 1.0
}

// Scrape method
func (ScrapeMemoryInfo) Scrape(ch chan<- prometheus.Metric, logger log.Logger) error {
	memMap := getMemoryInfo(logger)
	for memkey, memvalue := range memMap {
		newdesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, memoryinfo, "seconds_total"),
			"Seconds the memory info",
			[]string{"item"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(newdesc, prometheus.GaugeValue, memvalue, memkey)

	}
	return nil
}

// 获取/proc/meminfo，转化为字典
func getMemoryInfo(logger log.Logger) map[string]float64 {
	file, err := os.Open(procFilePath("meminfo"))
	if err != nil {
		level.Error(logger).Log("msg", "open file failed", "err", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line) //转成[]string 切片
		//Workaround for empty lines occasionally occur in CentOS 6.2 kernel 3.10.90.
		if len(parts) == 0 {
			continue
		}
		fv, err := strconv.ParseFloat(parts[1], 64) // str -> float64
		if err != nil {
			fmt.Println(err.Error())
		}
		key := parts[0][:len(parts[0])-1] // remove ":"
		key = reParens.ReplaceAllString(key, "_${1}")
		switch len(parts) {
		case 2: // no unit
		case 3: // has unit presume KB
			fv *= 1024
			key = key + "_bytes"
		default:
			level.Error(logger).Log("msg", "invalid line in meminfo", "line", line)
		}
		memInfo[key] = fv
	}
	return memInfo

}
