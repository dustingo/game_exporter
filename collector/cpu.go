package collector

import (
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"strconv"
	"sync"
)

type cpuCollector struct {
	fs  procfs.FS
	cpu *prometheus.Desc
	//cpuInfo      *prometheus.Desc
	cpuStats     []procfs.CPUStat
	logger       log.Logger
	cpuStatMutex sync.Mutex
}

const (
	linuxCpu = "linux_cpu_info"
)

type ScrapeCpuInfo struct{}

func (ScrapeCpuInfo) Name() string {
	return linuxCpu
}

func (ScrapeCpuInfo) Version() float64 {
	return 1.0
}
func (ScrapeCpuInfo) Help() string {
	return "Scrape Cpu info."
}

func (ScrapeCpuInfo) Scrape(ch chan<- prometheus.Metric, logger log.Logger) error {
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		return err
	}
	var c = &cpuCollector{
		fs:     fs,
		logger: logger,
		// dont need it
		//cpuInfo: prometheus.NewDesc("info",
		//	"CPU information from /proc/cpuinfo.",
		//	[]string{"package", "core", "cpu", "vendor", "family", "model", "model_name", "microcode", "stepping", "cachesize"}, nil,
		//),
		cpu: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, linuxCpu, "seconds_total"),
			"Seconds the CPUs spent in each mode.",
			[]string{"cpu", "mode"}, nil,
		),
	}
	stats, err := c.fs.Stat()
	if err != nil {
		return err
	}
	newStat := stats.CPU
	c.cpuStatMutex.Lock()
	defer c.cpuStatMutex.Unlock()
	for cpuID, cpuStat := range newStat {
		cpuNum := strconv.Itoa(cpuID)
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.User, cpuNum, "user")
		//ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Nice, cpuNum, "nice")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.System, cpuNum, "system")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Idle, cpuNum, "idle")
	}
	return nil
}
