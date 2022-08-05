package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	helmtime "helm.sh/helm/v3/pkg/time"
)

func TestCollector(t *testing.T) {
	defaultNamespace := "default"
	timestamp := helmtime.Unix(1000000000, 0).UTC()
	chartInfo := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:       "chickadee",
			Version:    "1.0.0",
			AppVersion: "0.0.1",
		},
	}
	releaseFixture := []*release.Release{
		{
			Name:      "foo",
			Version:   4,
			Namespace: defaultNamespace,
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusFailed,
			},
			Chart: chartInfo,
		},
		{
			Name:      "bar-pending-install",
			Version:   1,
			Namespace: defaultNamespace,
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusPendingInstall,
			},
			Chart: chartInfo,
		},
		{
			Name:      "bar-pending-rollback",
			Version:   1,
			Namespace: defaultNamespace,
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusPendingRollback,
			},
			Chart: chartInfo,
		},
		{
			Name:      "bar-pending-upgrade",
			Version:   1,
			Namespace: defaultNamespace,
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusPendingUpgrade,
			},
			Chart: chartInfo,
		},
		{
			Name:      "bar",
			Version:   1,
			Namespace: "othernamespace",
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusSuperseded,
			},
			Chart: chartInfo,
		},
		{
			Name:      "bar",
			Version:   2,
			Namespace: "othernamespace",
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusDeployed,
			},
			Chart: chartInfo,
		},
		{
			Name:      "foo-uninstalled",
			Version:   1,
			Namespace: defaultNamespace,
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusUninstalled,
			},
			Chart: chartInfo,
		},
		{
			Name:      "foo-uninstalling",
			Version:   1,
			Namespace: defaultNamespace,
			Info: &release.Info{
				LastDeployed: timestamp,
				Status:       release.StatusUninstalling,
			},
			Chart: chartInfo,
		},
	}

	// Init in-memory storage backend with release fixtures
	store := storage.Init(driver.NewMemory())
	for _, rel := range releaseFixture {
		if err := store.Create(rel); err != nil {
			t.Fatal(err)
		}
	}
	// Set namespace to all namespaces
	if mem, ok := store.Driver.(*driver.Memory); ok {
		mem.SetNamespace("")
	}
	// Build a mocked action config
	actionConfig := &action.Configuration{
		Releases:     store,
		KubeClient:   &kubefake.PrintingKubeClient{Out: ioutil.Discard},
		Capabilities: chartutil.DefaultCapabilities,
		Log:          func(format string, v ...interface{}) {},
	}

	metricsPath := "fixtures/all.metrics"
	exp, err := os.Open(metricsPath)
	if err != nil {
		t.Fatalf("Error opening fixture file: %v", err)
	}

	hc := NewHelmCollector(actionConfig)
	hc.TestRun = true
	if err := testutil.CollectAndCompare(hc, exp); err != nil {
		t.Error("Unexpected metrics returned:", err)
	}
}
