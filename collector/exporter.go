package collector

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

const (
	exporter = "exporter"
)

// Verify if Exporter implements prometheus.Collector
var _ prometheus.Collector = (*Exporter)(nil)

// Metric descriptors
var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "collector_duration_seconds"),
		"Collector time duration",
		[]string{"collector"}, nil,
	)
)

// Exporter collects game metrics. It implements prometheus.Collector.
type Exporter struct {
	ctx      context.Context
	logger   log.Logger
	scrapers []Scraper
	metrics  Metrics
}

// New returns a new game exporter for
func New(ctx context.Context, metrics Metrics, scrapers []Scraper, logger log.Logger) *Exporter {
	return &Exporter{
		ctx:      ctx,
		logger:   logger,
		scrapers: scrapers,
		metrics:  metrics,
	}
}

// Describe implement prometheus.Collector

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	//ch <- e.metrics.TotalScrapes.Desc()
	ch <- e.metrics.Error.Desc()
	e.metrics.ScrapeErrors.Describe(ch)
}

// Collect implement prometheus.Collector
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(e.ctx, ch)
	//ch <- e.metrics.TotalScrapes
	ch <- e.metrics.Error
	e.metrics.ScrapeErrors.Collect(ch)
}

// scrape go func() 执行各个collector的scrape
func (e *Exporter) scrape(ctx context.Context, ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, scraper := range e.scrapers {
		wg.Add(1)
		go func(scraper Scraper) {
			defer wg.Done()
			label := "collect." + scraper.Name()
			scrapeTime := time.Now()
			if err := scraper.Scrape(ch, log.With(e.logger, "scraper", scraper.Name())); err != nil {
				level.Error(e.logger).Log("msg", "Error from scraper", "scraper", scraper.Name(), "err", err)
				e.metrics.ScrapeErrors.WithLabelValues(label).Inc()
				e.metrics.Error.Set(1)
			}
			ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), label)
		}(scraper)
	}

}

// Metrics represents exporter metrics which values can be carried between http requests.
type Metrics struct {
	//TotalScrapes prometheus.Counter
	ScrapeErrors *prometheus.CounterVec
	Error        prometheus.Gauge
}

// NewMetrics create new metrics instance
func NewMetrics() Metrics {
	subsystem := exporter
	return Metrics{
		//TotalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
		//	Namespace: namespace,
		//	Subsystem: subsystem,
		//	Name:      "scrapes_total",
		//	Help:      "Total number of time game was scraped for metrics.",
		//}),
		ScrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occurred scraping a game.",
		}, []string{"collector"}),
		Error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from game resulted in an error (1 for error, 0 for success).",
		}),
	}
}
