package main

import (
	"context"
	"game_exporter/collector"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	listenAddress = kingpin.Flag(
		"web.listen-address",
		"Address to listen on for web interface and telemetry.",
	).Default(":9088").String()
	metricPath = kingpin.Flag(
		"web.telemetry-path",
		"Path under which to expose metrics.",
	).Default("/metrics").String()
	timeoutOffset = kingpin.Flag(
		"timeout-offset",
		"Offset to subtract from timeout in seconds.",
	).Default("0.25").Float64()
	configPath = kingpin.Flag(
		"config.path",
		"Path to .my.cnf file to read MySQL credentials from.",
	).Default("./gameporcess.yaml").String()
)

// scraper list all possible collection methods
var scrapers = map[collector.Scraper]bool{
	collector.ScrapeGameProcess{}:    true,
	collector.ScrapeCpuInfo{}:        true,
	collector.ScrapeFilesystemInfo{}: true,
}

func init() {
	prometheus.MustRegister(version.NewCollector("game_exporter"))
}

func newHandler(metrics collector.Metrics, scrapers []collector.Scraper, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filteredScrappers := scrapers
		params := r.URL.Query()["collecto[]"]
		// Use requst context for cancllation when connection gets closed
		ctx := r.Context()
		// if a timeout is configured via the Prometheus header,add it to the context
		if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
			timeoutSeconds, err := strconv.ParseFloat(v, 64)
			if err != nil {
				level.Error(logger).Log("msg", "Failed to parse timeout from Prometheus header", "err", err)
			} else {
				if *timeoutOffset >= timeoutSeconds {
					// Ignore timeout offset if it dosenot leave time to scrape
					level.Error(logger).Log("msg", "Timeout offset should be lower than prometheus scraope timeout", "offset", *timeoutOffset, "prometheus_scrape")
				} else {
					// Subtract timeout offset from timeout
					timeoutSeconds -= *timeoutOffset
				}
				// Create new timeout context with request context as parent
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds*float64((time.Second))))
				defer cancel()
				// Overwrite request with timeout context
				r = r.WithContext(ctx)
			}
		}
		level.Debug(logger).Log("msg", "collect[] params", "params", params)

		// Check if we have some collect[] query parameters
		if len(params) > 0 {
			filters := make(map[string]bool)
			for _, param := range params {
				filters[param] = true
			}
			filteredScrappers = nil
			for _, scraper := range scrapers {
				if filters[scraper.Name()] {
					filteredScrappers = append(filteredScrappers, scraper)
				}
			}
		}

		registry := prometheus.NewRegistry()
		registry.MustRegister(collector.New(ctx, metrics, filteredScrappers, logger))

		gatherers := prometheus.Gatherers{
			prometheus.DefaultGatherer,
			registry,
		}
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func main() {
	scraperFlags := map[collector.Scraper]*bool{}
	for scraper, enabledByDefault := range scrapers {
		defaultOn := "false"
		if enabledByDefault {
			defaultOn = "true"
		}

		f := kingpin.Flag(
			"collect."+scraper.Name(),
			scraper.Help(),
		).Default(defaultOn).Bool()

		scraperFlags[scraper] = f
	}
	// Parse flags
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("game_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	var landingPage = []byte(`<html>
<head><title>MySQLd exporter</title></head>
<body>
<h1>MySQLd exporter</h1>
<p><a href='` + *metricPath + `'>Metrics</a></p>
</body>
</html>
`)

	level.Info(logger).Log("msg", "Starting game_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", version.BuildContext())

	// Register only scrapers enabled by flag.
	enabledScrapers := []collector.Scraper{}
	for scraper, enabled := range scraperFlags {
		if *enabled {
			level.Info(logger).Log("msg", "Scraper enabled", "scraper", scraper.Name())
			enabledScrapers = append(enabledScrapers, scraper)
		}
	}
	handlerFunc := newHandler(collector.NewMetrics(), enabledScrapers, logger)
	http.Handle(*metricPath, promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, handlerFunc))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage)
	})

	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		os.Exit(1)
	}
}
