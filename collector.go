package main

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

const (
	metrics_prefix = "helm_release_"
)

var (
	commonLabels = []string{"name", "namespace"}
	status       = []release.Status{release.StatusDeployed, release.StatusFailed, release.StatusPendingInstall, release.StatusPendingRollback, release.StatusPendingUpgrade}
)

type cachedClient struct {
	*action.List
	releases []*release.Release
	mutex    sync.Mutex
	ttl      time.Time
}

type helmCollector struct {
	client       *cachedClient
	TestRun      bool
	Info         *prometheus.Desc
	Revision     *prometheus.Desc
	Status       *prometheus.Desc
	Updated      *prometheus.Desc
	ListDuration *prometheus.HistogramVec
}

func NewHelmCollector(cfg *action.Configuration) *helmCollector {
	// Setup a helm list action client
	client := action.NewList(cfg)
	client.AllNamespaces = true
	client.Deployed = true
	client.Failed = true
	client.Pending = true
	client.SetStateMask()

	listDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: metrics_prefix + "list_duration_seconds",
		Help: "helm list latency distribution"},
		[]string{"status"},
	)

	return &helmCollector{
		client: &cachedClient{List: client},
		Info: prometheus.NewDesc(
			metrics_prefix+"info",
			"Information about helm release",
			append(commonLabels, "chart", "chart_version", "app_version"),
			prometheus.Labels{},
		),
		Revision: prometheus.NewDesc(
			metrics_prefix+"revision",
			"Currently deployed helm chart revision",
			commonLabels,
			prometheus.Labels{},
		),
		Status: prometheus.NewDesc(
			metrics_prefix+"status",
			"Status of a helm release",
			append(commonLabels, "status"),
			prometheus.Labels{},
		),
		Updated: prometheus.NewDesc(
			metrics_prefix+"updated",
			"Release update Unix time",
			commonLabels,
			prometheus.Labels{},
		),
		ListDuration: listDuration,
	}
}

func (hc *helmCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- hc.Info
	ch <- hc.Revision
	ch <- hc.Status
	ch <- hc.Updated
	hc.ListDuration.MetricVec.Describe(ch)
}

func (hc *helmCollector) Collect(ch chan<- prometheus.Metric) {
	runStart := time.Now()
	results, cached, err := hc.client.ListReleases()
	runDurationSeconds := time.Since(runStart).Seconds()
	if hc.TestRun {
		runDurationSeconds = 0.18
	}
	if err != nil {
		log.Println(err)
		ch <- prometheus.NewInvalidMetric(hc.Info, err)
		ch <- prometheus.NewInvalidMetric(hc.Revision, err)
		ch <- prometheus.NewInvalidMetric(hc.Status, err)
		ch <- prometheus.NewInvalidMetric(hc.Updated, err)
		if !cached {
			hc.ListDuration.WithLabelValues("error").Observe(runDurationSeconds)
		}
		return
	}
	if !cached {
		hc.ListDuration.WithLabelValues("success").Observe(runDurationSeconds)
	}
	hc.ListDuration.MetricVec.Collect(ch)

	for _, r := range results {
		ch <- prometheus.MustNewConstMetric(
			hc.Info,
			prometheus.GaugeValue,
			1.0,
			r.Name,
			r.Namespace,
			formatChartName(r.Chart),
			formatChartVersion(r.Chart),
			formatAppVersion(r.Chart),
		)

		ch <- prometheus.MustNewConstMetric(
			hc.Revision,
			prometheus.GaugeValue,
			float64(r.Version),
			r.Name,
			r.Namespace,
		)

		// Send one metric per status
		rStatus := r.Info.Status
		for _, s := range status {
			value := 0.0
			if s == rStatus {
				value = 1.0
			}
			ch <- prometheus.MustNewConstMetric(
				hc.Status,
				prometheus.GaugeValue,
				value,
				r.Name,
				r.Namespace,
				s.String(),
			)
		}

		ch <- prometheus.MustNewConstMetric(
			hc.Updated,
			prometheus.GaugeValue,
			float64(r.Info.LastDeployed.Unix()),
			r.Name,
			r.Namespace,
		)
	}
}

func (c *cachedClient) ListReleases() ([]*release.Release, bool, error) {
	c.mutex.Lock()
	cached := true
	defer c.mutex.Unlock()
	if time.Now().After(c.ttl) {
		// Update the cache
		cached = false
		r, err := c.Run()
		if err != nil {
			// Don't return the cached releases in case of error
			return nil, cached, err
		}
		c.releases = r
		// Cache will be dirty in 1 minute from now
		c.ttl = time.Now().Add(time.Duration(1 * time.Minute))
	}

	return c.releases, cached, nil
}

func issue1347(c *chart.Chart) error {
	if c == nil || c.Metadata == nil {
		// This is an edge case that has happened in prod, though we don't
		// know how: https://github.com/helm/helm/issues/1347
		return errors.New("issue1347")
	}
	return nil
}

func formatChartName(c *chart.Chart) string {
	if err := issue1347(c); err != nil {
		return "MISSING"
	}
	return c.Name()
}

func formatChartVersion(c *chart.Chart) string {
	if err := issue1347(c); err != nil {
		return "MISSING"
	}
	return c.Metadata.Version
}

func formatAppVersion(c *chart.Chart) string {
	if err := issue1347(c); err != nil {
		return "MISSING"
	}
	return c.AppVersion()
}
