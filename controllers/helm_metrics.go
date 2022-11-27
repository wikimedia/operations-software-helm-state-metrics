/*
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
	"github.com/prometheus/client_golang/prometheus"
	"helm.sh/helm/v3/pkg/release"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	metricsPrefix = "helm_release_"
	commonLabels  = []string{"name", "namespace"}
	status        = []release.Status{
		release.StatusDeployed,
		release.StatusFailed,
		release.StatusPendingInstall,
		release.StatusPendingRollback,
		release.StatusPendingUpgrade,
		// The following additional states are ignored on purpose as it does not
		// seem like adding metrics for them would add additional value.
		//release.StatusUnknown,
		//release.StatusUninstalled,
		//release.StatusSuperseded,
		//release.StatusUninstalling,
	}

	metricInfo     *prometheus.GaugeVec
	metricRevision *prometheus.GaugeVec
	metricStatus   *prometheus.GaugeVec
	metricUpdated  *prometheus.GaugeVec
	metricErrors   *prometheus.CounterVec
)

func init() {
	metricInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricsPrefix + "info",
		Help: "Information about helm release",
	}, append(commonLabels, "chart", "chart_version", "app_version", "revision"))

	metricRevision = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricsPrefix + "revision",
		Help: "Currently deployed helm chart revision"},
		commonLabels)

	metricStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricsPrefix + "status",
		Help: "Status of a helm release"},
		append(commonLabels, "status"))

	metricUpdated = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricsPrefix + "updated",
		Help: "Release update Unix time"},
		commonLabels)

	metricErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: metricsPrefix + "errors",
		Help: "Errors occurred during metrics generation per namespace"},
		[]string{"namespace"})

	metrics.Registry.MustRegister(
		metricInfo,
		metricRevision,
		metricStatus,
		metricUpdated,
		metricErrors,
	)
}
