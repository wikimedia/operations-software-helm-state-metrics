/*
Copyright The Helm Authors.
Copyright 2022 - Janis Meybohm, Wikimedia Foundation Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"helm.sh/helm/v3/pkg/chart"
)

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

// getGaugeValue reads the value currently stored for a given GaugeVec and label combination from the prometheus registry
func getGaugeValue(metric *prometheus.GaugeVec, lvs ...string) (float64, error) {
	var m = &dto.Metric{}
	if err := metric.WithLabelValues(lvs...).Write(m); err != nil {
		return 0, err
	}
	return m.Gauge.GetValue(), nil
}
