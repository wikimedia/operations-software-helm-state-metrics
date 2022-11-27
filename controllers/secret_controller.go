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
	"context"
	"errors"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	helmStorageDriver "helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Regex to extract the helm release name and revision from the secret name (in case of deletions)
var reReleaseName = regexp.MustCompile(`^sh\.helm\.release\.v\d+\.([^\.]+)\.v(\d+)$`)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// It is used here to collect metrics about Helm releases in the cluster.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	helmRelease := helmStorageDriver.NewSecrets(NewSecretsClient(r.Client, req.Namespace))
	release, err := helmRelease.Get(req.Name)
	if err != nil {
		if err == helmStorageDriver.ErrReleaseNotFound {
			// In case of deleted releases we can only reconstruct the name and revision from the
			// name of the secret object (as the object itself does no longer exist).
			//
			// Parse releaseName and releaseRevision from the secret name in order to clean up old
			// metrics from the prometheus registry if release revisions or complete releases are
			// deleted.
			match := reReleaseName.FindStringSubmatch(req.Name)
			if match == nil {
				log.Error(errors.New("unable to parse release name from secret name"), "Regex did not match")
				metricErrors.WithLabelValues(req.Namespace).Inc()
				return ctrl.Result{}, nil
			}
			releaseName := match[1]
			releaseRevision, err := strconv.Atoi(match[2])
			if err != nil {
				log.Error(err, "Unable to parse revision from secret name")
				return ctrl.Result{}, err
			}

			// Get the latest release observed from the prometheus registry
			latestSeenReleaseRevision, err := getGaugeValue(metricRevision, releaseName, req.Namespace)
			if err != nil {
				log.Error(err, "Unable to get release revision from registry")
				metricErrors.WithLabelValues(req.Namespace).Inc()
				return ctrl.Result{}, err
			}

			// If a old revision is deleted, it's fine to just remove it's metricsInfo entry.
			// If the latest revision is deleted this means the full release has been deleted, so
			// all metrics need to be removed.
			metricInfo.DeletePartialMatch(prometheus.Labels{"name": releaseName, "namespace": req.Namespace, "revision": strconv.Itoa(releaseRevision)})
			if releaseRevision == int(latestSeenReleaseRevision) {
				// latest release was deleted, clean up metrics
				genericLabels := prometheus.Labels{"name": releaseName, "namespace": req.Namespace}
				metricRevision.DeletePartialMatch(genericLabels)
				metricStatus.DeletePartialMatch(genericLabels)
				metricUpdated.DeletePartialMatch(genericLabels)
			}
			return ctrl.Result{}, nil
		}
		log.Error(err, "Unable to get release")
		metricErrors.WithLabelValues(req.Namespace).Inc()
		return ctrl.Result{}, err
	}

	chartName := formatChartName(release.Chart)
	chartVersion := formatChartVersion(release.Chart)
	appVersion := formatAppVersion(release.Chart)
	releaseRevision := float64(release.Version)
	genericLabels := prometheus.Labels{"name": release.Name, "namespace": req.Namespace}
	log = log.WithValues("namespace", req.Namespace, "release", release.Name, "chart", chartName, "chartVersion", chartVersion, "revision", releaseRevision, "status", release.Info.Status)

	// Get the latest release observed from the prometheus registry
	latestSeenReleaseRevision, err := getGaugeValue(metricRevision, release.Name, req.Namespace)
	if err != nil {
		log.Error(err, "unable to get release revision from registry")
		metricErrors.WithLabelValues(req.Namespace).Inc()
		return ctrl.Result{}, err
	}
	if releaseRevision < latestSeenReleaseRevision {
		log.WithValues("latestSeenReleaseRevision", latestSeenReleaseRevision).Info("Skipping as we've already seen a newer release")
		return ctrl.Result{}, nil
	}
	if latestSeenReleaseRevision > 0.0 {
		// This is a newer revision for an existing release, delete old info metric
		metricInfo.DeletePartialMatch(genericLabels)
	}

	// Update the metrics in prometheus registry
	metricInfo.WithLabelValues(release.Name, req.Namespace,
		chartName,
		chartVersion,
		appVersion,
		strconv.Itoa(release.Version)).Set(1.0)
	metricRevision.With(genericLabels).Set(releaseRevision)
	metricUpdated.With(genericLabels).Set(float64(release.Info.LastDeployed.Unix()))
	// Send one metric per status
	for _, s := range status {
		value := 0.0
		if s == release.Info.Status {
			value = 1.0
		}
		metricStatus.WithLabelValues(release.Name, req.Namespace, s.String()).Set(value)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Filter for objects with "owner: helm" label
	pred, err := predicate.LabelSelectorPredicate(v1.LabelSelector{MatchLabels: map[string]string{"owner": "helm"}})
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(pred).
		Complete(r)
}
