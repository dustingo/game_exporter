package collector

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// MyConfig config结构体 ，对应yaml的process_name
type MyConfig struct {
	Processnames []Info `yaml:"process_names"`
}

// Info 结构体，对应process_names下的-name和cmdline
type Info struct {
	Name    string   `yaml:"name"`
	Cmdline []string `yaml:"cmdline"`
}

const (
	// subsystem
	gameProcess = "game_linux_process_num"
)

// GetConfig 解析yaml，返回myconfig结构体指针
func GetConfig() (*MyConfig, error) {
	var fileName string
	for _, para := range os.Args {
		if strings.HasPrefix(para, "--config.path") {
			//fmt.Println(para)
			//fmt.Printf("Type:%T\n",para)
			fileName = string(strings.Split(para, "=")[1])
		}
	}
	if fileName == "" {
		fileName = "./gameprocess.yaml"
	}
	var myconfig = new(MyConfig)
	yamlInfo, _ := ioutil.ReadFile(fileName)
	err := yaml.Unmarshal(yamlInfo, myconfig)
	return myconfig, err
}

// 将结构体内cmdline中，涉及到“/”全部添加转义符 “\”
func modifyString(s []string) []string {
	// 只限于当cmdline有两个元素的时候，才去替换
	if len(s) == 2 {
		for i := 0; i < len(s); i++ {
			s[i] = strings.Replace(s[i], "/", "\\/", -1)
		}
	}
	return s
}

// ScrapeGameProcess collects
type ScrapeGameProcess struct{}

// Name of the Scraper Unique
func (ScrapeGameProcess) Name() string {
	return gameProcess
}

// Version of config whitch scraper is avaliable
func (ScrapeGameProcess) Version() float64 {
	return 1.0
}
func (ScrapeGameProcess) Help() string {
	return "scrape the number of game processes"
}
func (ScrapeGameProcess) Scrape(ch chan<- prometheus.Metric, logger log.Logger) error {
	var cmd string
	processNumData := make(map[string]int)
	configStruct, err := GetConfig()
	if err != nil {
		return err
	}
	for _, v := range configStruct.Processnames {

		if len(v.Cmdline) == 1 {
			cmd = `ps aux | awk '/` + v.Cmdline[0] + `/ && !/awk/ '|wc -l`
		} else {
			newcmdline := modifyString(v.Cmdline)
			cmd = `ps aux | awk '/` + newcmdline[0] + `/ && /` + newcmdline[1] + `/  && !/awk/ '|wc -l`
		}
		result, err := exec.Command("/bin/bash", "-c", cmd).Output()
		if err != nil {
			fmt.Println(err.Error())
		}
		pronum, _ := strconv.Atoi(strings.TrimSuffix(string(result), "\n"))
		processNumData[v.Name] = pronum

	}
	for procName, procNum := range processNumData {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(gameProcess, "number of process in yaml config", []string{"procname"}, nil),
			prometheus.GaugeValue,
			float64(procNum),
			procName,
		)
	}
	return nil
}

var _ Scraper = ScrapeGameProcess{}
