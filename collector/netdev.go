package collector

import (
	"bufio"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	procNetDevInterfaceRE = regexp.MustCompile(`^(.+): *(.+)$`)
	procNetDevFieldSep    = regexp.MustCompile(` +`)
	ignoreDevice          = regexp.MustCompile("tap.*|veth.*|br.*|docker.*|virbr*|lo*") // 忽略的设备
	acceptDevice          = regexp.MustCompile("eh*|ens*")                              // 需要抓取的设备
)

// 网卡状态字典
type netDevStats map[string]map[string]uint64

type ScrapeNetInfo struct{}

const (
	linuxNet = "linux_net_info"
)

// Name method
func (ScrapeNetInfo) Name() string {
	return linuxNet
}

// Version
func (ScrapeNetInfo) Version() float64 {
	return 1.0
}

// Help
func (ScrapeNetInfo) Help() string {
	return "Scrape linux network receive and transmit info"
}

func (ScrapeNetInfo) Scrape(ch chan<- prometheus.Metric, logger log.Logger) error {
	netInfo, err := getNetDevStats(ignoreDevice, acceptDevice, logger)
	if err != nil {
		return err
	}
	for devName, netBytes := range netInfo {
		receiveDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, linuxNet, "receive_bytes_total"),
			"Network device statistic receive_bytes",
			[]string{"device"}, nil,
		)
		transmitDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, linuxNet, "transmit_bytes_total"),
			"Network device statistic transmit_bytes",
			[]string{"device"}, nil,
		)

		ch <- prometheus.MustNewConstMetric(receiveDesc, prometheus.GaugeValue, float64(netBytes["receive_bytes"]), devName)
		ch <- prometheus.MustNewConstMetric(transmitDesc, prometheus.GaugeValue, float64(netBytes["transmit_bytes"]), devName)
	}
	return nil
}

func getNetDevStats(ignore *regexp.Regexp, accept *regexp.Regexp, logger log.Logger) (netDevStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		fmt.Println(err.Error())
	}

	defer file.Close()
	return parseNetDevStats(file, ignore, accept, logger)
}

// 实际抓取
func parseNetDevStats(r io.Reader, ignore *regexp.Regexp, accept *regexp.Regexp, logger log.Logger) (netDevStats, error) {
	// parse
	scanner := bufio.NewScanner(r)
	scanner.Scan() // skip first header
	scanner.Scan()
	parts := strings.Split(scanner.Text(), "|")
	if len(parts) != 3 { // interface | receive | transmit
		return nil, fmt.Errorf("invalid header line in net/dev:%s", scanner.Text())
	}

	receiveHeader := strings.Fields(parts[1])
	transmitHeader := strings.Fields(parts[2])
	headerLength := len(receiveHeader) + len(transmitHeader)

	netDev := netDevStats{}
	for scanner.Scan() {
		line := strings.TrimLeft(scanner.Text(), " ")
		parts := procNetDevInterfaceRE.FindStringSubmatch(line)
		if len(parts) != 3 {
			return nil, fmt.Errorf("couldnt get interface name,invalid line in net/dev:%q", line)
		}
		dev := parts[1]
		if ignore != nil && ignore.MatchString(dev) {
			level.Info(logger).Log("msg", "Ingnoring device", "device", dev)
			continue
		}
		if accept != nil && !accept.MatchString(dev) {
			level.Info(logger).Log("msg", "Ingoring device", "device", dev)
			continue
		}
		values := procNetDevFieldSep.Split(strings.TrimLeft(parts[2], " "), -1)
		if len(values) != headerLength {
			return nil, fmt.Errorf("msg", "could not get values,invalid line in net/dev：%q", parts[2])
		}
		devStats := map[string]uint64{}
		addStats := func(key, value string) {
			v, err := strconv.ParseUint(value, 0, 64)
			if err != nil {
				level.Info(logger).Log("msg", "invalid value in netstats", "key", key, "value", value, "err", err)
				return
			}
			devStats[key] = v
		}

		for i := 0; i < len(receiveHeader); i++ {
			addStats("receive_"+receiveHeader[i], values[i])
		}
		for i := 0; i < len(transmitHeader); i++ {
			addStats("transmit_"+transmitHeader[i], values[i+len(receiveHeader)])
		}
		netDev[dev] = devStats
	}
	return netDev, scanner.Err()
}
