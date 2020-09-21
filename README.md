#### game_exporter
- 说明：  
本exporter架构由mysqld_exporter基础架构修改而来
- 目的：  
  1.摆脱繁杂(更专业)的指标  
  2.专注于游戏运维中要关注的指标  
  3.便于监控游戏进程  
  4.便于增加collector,目前collector包含cpu和游戏进程,内存，文件系统，网络接收、发送量，系统平均负载，其他必要指标正在过滤中。

- 使用:    
./game_exporter --config.path=gameprocess.yaml  
or  
./game_exporter   
or  
systemd   
gameprocess.yaml 为游戏进程配置文件  
name为进程名  
cmdline为定位进程所需的字段，最大只能两条
- 增加新的collector:  
创建新的collector只需要在collector中实现此接口并在game_exporter.go中注册即可
```golang
type Scraper interface {
	Name() string
	Help() string
	Version() float64
	Scrape(ch chan<- prometheus.Metric,logger log.Logger) error
}
```
- 去除默认metrics：  
注释 pkg\mod\github.com\prometheus\client_golang@v1.7.1\prometheus\registry.go中的init
```golang
func init() {
	//MustRegister(NewProcessCollector(ProcessCollectorOpts{}))
	//MustRegister(NewGoCollector())
}
```
- 修改log为CST时间:
修改 pkg\mod\github.com\prometheus\common@v0.10.0\promlog\log.go
```golang
将
func() time.Time { return time.Now().UTC() },
修改为
func() time.Time { return time.Now().Local() },
```
- 各指标metric
   - cpu: game_linux_cpu_info_seconds_total
   - mem: game_memory_info_seconds_total
   - filesystem: game_linux_filesystem_info_total_free
   - network: game_linux_net_info_receive_bytes_total|game_linux_net_info_transmit_bytes_total
   - laodavg: game_linux_load_avg1|game_linux_load_avg5|game_linux_load_avg15
   - process: game_linux_process_num