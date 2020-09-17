package collector

import (
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Scraper interface {
	Name() string
	Help() string
	Version() float64
	Scrape(ch chan<- prometheus.Metric,logger log.Logger) error
}
