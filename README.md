#### game_exporter
- 说明：  
本exporter架构由mysqld_exporter基础架构修改而来
- 目的：  
  1.摆脱繁杂(更专业)的指标  
  2.专注于游戏运维中要关注的指标  
  3.便于监控游戏进程  
  4.便于增加collector,目前只有cpu和游戏进程监控，其他必要指标正在过滤中。

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
创建新的collector只需要在collector中实现此接口即可
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

