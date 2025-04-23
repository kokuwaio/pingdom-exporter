package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/monotek/pingdom-exporter/pkg/pingdom-exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// VERSION will hold the version number injected during the build.
	VERSION string

	defaultUptimeSLO  float64
	metricsPath       string
	outageCheckPeriod int
	parserTags        bool
	port              int
	tagFormat         string
	tags              string
	tagExtraLabels    string
	token             string

	extraLabelsMap      pingdom.ExtraLabel
	extraLabelsMapOrder []string
	isExtraLabels       bool

	pingdomCheckAvailableErrorBudgetDesc *prometheus.Desc
	pingdomCheckErrorBudgetDesc          *prometheus.Desc
	pingdomCheckResponseTimeDesc         *prometheus.Desc
	pingdomCheckStatusDesc               *prometheus.Desc
	pingdomCheckTags                     *prometheus.Desc
	pingdomCheckTagsLabel                *prometheus.Desc
	pingdomDownTimeDesc                  *prometheus.Desc
	pingdomOutageCheckPeriodDesc         *prometheus.Desc
	pingdomOutagesDesc                   *prometheus.Desc
	pingdomUpDesc                        *prometheus.Desc
	pingdomUpTimeDesc                    *prometheus.Desc
)

func init() {
	flag.IntVar(&port, "port", 9158, "port to listen on")
	flag.IntVar(&outageCheckPeriod, "outage-check-period", 7, "time (in days) in which to retrieve outage data from the Pingdom API")
	flag.Float64Var(&defaultUptimeSLO, "default-uptime-slo", 99.0, "default uptime SLO to be used when the check doesn't provide a uptime SLO tag (i.e. uptime_slo_999 to 99.9% uptime SLO)")
	flag.StringVar(&metricsPath, "metrics-path", "/metrics", "path under which to expose metrics")
	flag.StringVar(&tags, "tags", "", "tag list separated by commas")
	flag.BoolVar(&parserTags, "parser-tags", false, "Enable tag formatting based on a regular expression")
	flag.StringVar(&tagFormat, "tag-format", "^([a-zA-Z0-9_]+):(.+)$", "Regular expression used to format tags")
	flag.StringVar(&tagExtraLabels, "tag-extra-labels", "", "allows adding custom labels to metrics, ensuring they follow a consistent format with the label_ prefix. Requires -parser-tags=true")
}

type pingdomCollector struct {
	client *pingdom.Client
}

func (pc pingdomCollector) Describe(ch chan<- *prometheus.Desc) {
	pingdomCheckAvailableErrorBudgetDesc = prometheus.NewDesc(
		"pingdom_uptime_slo_error_budget_available_seconds",
		"Number of seconds of downtime we can still have without breaking the uptime SLO",
		[]string{"id", "name", "hostname"}, nil,
	)

	pingdomCheckErrorBudgetDesc = prometheus.NewDesc(
		"pingdom_uptime_slo_error_budget_total_seconds",
		"Maximum number of allowed downtime, in seconds, according to the uptime SLO",
		[]string{"id", "name", "hostname"}, nil,
	)

	pingdomCheckResponseTimeDesc = prometheus.NewDesc(
		"pingdom_uptime_response_time_seconds",
		"The response time of last test, in seconds",
		[]string{"id", "name", "hostname", "resolution", "paused"}, nil,
	)

	pingdomCheckStatusDesc = prometheus.NewDesc(
		"pingdom_uptime_status",
		"The current status of the check (1: up, 0: down)",
		[]string{"id", "name", "hostname", "resolution", "paused"}, nil,
	)

	pingdomCheckTags = prometheus.NewDesc(
		"pingdom_tags",
		"The current tags of the check",
		[]string{"id", "name", "type", "count"}, nil,
	)

	if isExtraLabels {
		pingdomCheckTagsLabel = prometheus.NewDesc(
			"pingdom_tags_label",
			"The current tags of the check",
			append([]string{"id"}, pingdom.GetLabelNamesFromExtraLabels(extraLabelsMap)...), nil,
		)
	} else {
		pingdomCheckTagsLabel = prometheus.NewDesc(
			"pingdom_tags_label",
			"The current tags of the check",
			[]string{"id", "label_key", "label_value", "name"}, nil,
		)
	}

	pingdomDownTimeDesc = prometheus.NewDesc(
		"pingdom_down_seconds",
		"Total down time within the outage check period, in seconds",
		[]string{"id", "name", "hostname"}, nil,
	)

	pingdomOutageCheckPeriodDesc = prometheus.NewDesc(
		"pingdom_slo_period_seconds",
		"Outage check period, in seconds",
		nil, nil,
	)

	pingdomOutagesDesc = prometheus.NewDesc(
		"pingdom_outages_total",
		"Number of outages within the outage check period",
		[]string{"id", "name", "hostname"}, nil,
	)

	pingdomUpDesc = prometheus.NewDesc(
		"pingdom_up",
		"Whether the last pingdom scrape was successfull (1: up, 0: down).",
		nil, nil,
	)

	pingdomUpTimeDesc = prometheus.NewDesc(
		"pingdom_up_seconds",
		"Total up time within the outage check period, in seconds",
		[]string{"id", "name", "hostname"}, nil,
	)

	ch <- pingdomCheckAvailableErrorBudgetDesc
	ch <- pingdomCheckErrorBudgetDesc
	ch <- pingdomCheckResponseTimeDesc
	ch <- pingdomCheckStatusDesc
	ch <- pingdomCheckTags
	ch <- pingdomCheckTagsLabel
	ch <- pingdomDownTimeDesc
	ch <- pingdomOutageCheckPeriodDesc
	ch <- pingdomOutagesDesc
	ch <- pingdomUpDesc
	ch <- pingdomUpTimeDesc
}

func (pc pingdomCollector) Collect(ch chan<- prometheus.Metric) {
	outageCheckPeriodDuration := time.Hour * time.Duration(24*outageCheckPeriod)
	outageCheckPeriodSecs := float64(outageCheckPeriodDuration / time.Second)

	checks, err := pc.client.Checks.List(map[string]string{
		"include_tags": "true",
		"tags":         pc.client.Tags,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting checks: %v", err)
		ch <- prometheus.MustNewConstMetric(
			pingdomUpDesc,
			prometheus.GaugeValue,
			float64(0),
		)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		pingdomUpDesc,
		prometheus.GaugeValue,
		float64(1),
	)

	ch <- prometheus.MustNewConstMetric(
		pingdomOutageCheckPeriodDesc,
		prometheus.GaugeValue,
		outageCheckPeriodSecs,
	)

	var wg sync.WaitGroup

	for _, check := range checks {
		// Ignore this check based on the presence of the ignore label
		if check.HasIgnoreTag() {
			continue
		}

		id := strconv.Itoa(check.ID)
		tags := check.AllTags()
		resolution := strconv.Itoa(check.Resolution)

		var status float64
		paused := "false"
		switch check.Status {
		case "paused":
			paused = "true"
		case "up":
			status = 1
		}

		tagExtraLabelsValuesMap := make(map[string]string)

		for _, tag := range tags {
			if parserTags {
				tagLabel, _ := pingdom.TagLabel(tag.Name, tagFormat)
				if isExtraLabels {
					tagExtraLabelsValuesMap[tagLabel.LabelKey] = tagLabel.LabelValue
				} else {
					ch <- prometheus.MustNewConstMetric(
						pingdomCheckTagsLabel,
						prometheus.GaugeValue,
						float64(tagLabel.Formatted),
						id,
						tagLabel.LabelKey,
						tagLabel.LabelValue,
						tag.Name,
					)
				}
			} else {
				ch <- prometheus.MustNewConstMetric(
					pingdomCheckTags,
					prometheus.GaugeValue,
					float64(1),
					id,
					tag.Name,
					tag.Type,
					fmt.Sprint(tag.Count),
				)
			}
		}

		if len(tagExtraLabelsValuesMap) > 0 {
			ch <- prometheus.MustNewConstMetric(
				pingdomCheckTagsLabel,
				prometheus.GaugeValue,
				float64(1),
				append([]string{id}, pingdom.GetExtraLabelsValues(tagExtraLabelsValuesMap, extraLabelsMap, extraLabelsMapOrder)...)...,
			)
		}

		ch <- prometheus.MustNewConstMetric(
			pingdomCheckStatusDesc,
			prometheus.GaugeValue,
			status,
			id,
			check.Name,
			check.Hostname,
			resolution,
			paused,
		)

		ch <- prometheus.MustNewConstMetric(
			pingdomCheckResponseTimeDesc,
			prometheus.GaugeValue,
			float64(check.LastResponseTime)/1000.0,
			id,
			check.Name,
			check.Hostname,
			resolution,
			paused,
		)

		// Retrieve outages for check
		var downCount, upTime, downTime float64

		// Maximum allowed downtime, in seconds, according to the uptime SLO
		uptimeErrorBudget := outageCheckPeriodSecs * (100.0 - check.UptimeSLOFromTags(defaultUptimeSLO)) / 100.0

		// Retrieve the outage list within the desired period for this check, in background
		wg.Add(1)

		go func(check pingdom.CheckResponse) {
			defer wg.Done()

			// Retrieve the list of outages within the outage period for the given check
			now := time.Now()
			states, err := pc.client.OutageSummary.List(check.ID, map[string]string{
				"from": strconv.FormatInt(now.Add(-outageCheckPeriodDuration).Unix(), 10),
				"to":   strconv.FormatInt(now.Unix(), 10),
			})

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting outages for check %d: %v", check.ID, err)
				return
			}

			for _, state := range states {
				switch state.Status {
				case "down":
					downCount++
					downTime = downTime + float64(state.ToTime-state.FromTime)
				case "up":
					upTime = upTime + float64(state.ToTime-state.FromTime)
				}
			}

			ch <- prometheus.MustNewConstMetric(
				pingdomOutagesDesc,
				prometheus.GaugeValue,
				downCount,
				id,
				check.Name,
				check.Hostname,
			)

			ch <- prometheus.MustNewConstMetric(
				pingdomUpTimeDesc,
				prometheus.GaugeValue,
				upTime,
				id,
				check.Name,
				check.Hostname,
			)

			ch <- prometheus.MustNewConstMetric(
				pingdomDownTimeDesc,
				prometheus.GaugeValue,
				downTime,
				id,
				check.Name,
				check.Hostname,
			)

			ch <- prometheus.MustNewConstMetric(
				pingdomCheckErrorBudgetDesc,
				prometheus.GaugeValue,
				uptimeErrorBudget,
				id,
				check.Name,
				check.Hostname,
			)

			ch <- prometheus.MustNewConstMetric(
				pingdomCheckAvailableErrorBudgetDesc,
				prometheus.GaugeValue,
				uptimeErrorBudget-downTime,
				id,
				check.Name,
				check.Hostname,
			)
		}(check)
	}

	wg.Wait()
}

func main() {
	var client *pingdom.Client
	flag.Parse()

	token = os.Getenv("PINGDOM_API_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Pingdom API token must be provided via the PINGDOM_API_TOKEN environment variable, exiting")
		os.Exit(1)
	}

	client, err := pingdom.NewClientWithConfig(pingdom.ClientConfig{
		Token: token,
		Tags:  tags,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot create Pingdom client, exiting")
		os.Exit(1)
	}

	if parserTags && tagExtraLabels != "" {
		extraLabelsMap, extraLabelsMapOrder = pingdom.ProcessExtraLabels(tagExtraLabels)
		isExtraLabels = len(extraLabelsMap) > 0
	}

	registry := prometheus.NewPedanticRegistry()
	collector := pingdomCollector{
		client: client,
	}

	registry.MustRegister(
		collector,
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	http.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
			 <head><title>Pingdom Exporter</title></head>
			 <body>
			 <h1>Pingdom Exporter</h1>
			 <p><a href='` + metricsPath + `'>Metrics</a></p>
			 </body>
			 </html>`))
		if err != nil {
			http.Error(w, "Unable to write response", http.StatusInternalServerError)
		}
	})

	fmt.Fprintf(os.Stdout, "Pingdom Exporter %v listening on http://0.0.0.0:%v\n", VERSION, port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           nil,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
