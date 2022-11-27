package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"helm.sh/helm/v3/pkg/chart"
	rspb "helm.sh/helm/v3/pkg/release"
	helmStorageDriver "helm.sh/helm/v3/pkg/storage/driver"
	helmtime "helm.sh/helm/v3/pkg/time"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func newUnicorn(name, namespace, chartName, version, appVersion string, revision int, status rspb.Status) (string, *rspb.Release) {
	lastDeployed, _ := helmtime.Parse(time.RFC3339, "2001-01-15T19:29:00Z")
	secretName := fmt.Sprintf("sh.helm.release.v1.%s.v%d", name, revision)
	return secretName, &rspb.Release{
		Name:      name,
		Namespace: namespace,
		Version:   revision,
		Info: &rspb.Info{
			LastDeployed: lastDeployed,
			Status:       status,
		},
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:       chartName,
				Version:    version,
				AppVersion: appVersion,
			},
		},
	}
}

var _ = Describe("Secret controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)
	var (
		RelevantMetricNames = []string{metricsPrefix + "info", metricsPrefix + "revision", metricsPrefix + "status", metricsPrefix + "updated", metricsPrefix + "errors"}
	)

	// Contrary to what one might believe these stages are not independent of each other as they
	// all share the same kube-apiserver instance and it's state!
	When("There are not helm releases", func() {
		It("exports no helm_release metrics", func() {
			Expect(testutil.GatherAndCount(metrics.Registry, RelevantMetricNames...)).Should(Equal(0))
		})
	})
	When("There is a helm release", func() {
		var (
			namespace, secretName string
			release               *rspb.Release
			helmReleaseClient     *helmStorageDriver.Secrets
		)
		BeforeEach(func() {
			namespace = "default"
			helmReleaseClient = helmStorageDriver.NewSecrets(NewSecretsClient(k8sClient, namespace))
			secretName, release = newUnicorn("punkunicorn", namespace, "unicorn", "0.0.1", "1.1.0", 1, rspb.StatusDeployed)
		})
		It("exports proper helm_release metrics (./fixtures/one_release.metrics)", func() {
			By("Creating a helm release")
			Expect(helmReleaseClient.Create(secretName, release)).Should(Succeed())

			Eventually(func() (int, error) {
				return testutil.GatherAndCount(metrics.Registry, RelevantMetricNames...)
			}, timeout, interval).Should(Equal(8))

			metricsPath := "../fixtures/one_release.metrics"
			exp, err := os.Open(metricsPath)
			Expect(err).NotTo(HaveOccurred())
			defer exp.Close()

			// I don't know why exactly but without checking with GatherAndCount first this constantly
			// fails even with increased timeout/internal.
			Eventually(func() error {
				return testutil.GatherAndCompare(metrics.Registry, exp, RelevantMetricNames...)
			}, timeout, interval).Should(Succeed())

		})
		It("does no longer export metrics after deleting the release", func() {
			helmReleaseClient.Delete(secretName)
			Eventually(func() (int, error) {
				return testutil.GatherAndCount(metrics.Registry, RelevantMetricNames...)
			}, timeout, interval).Should(Equal(0))
		})
	})
	When("There are multiple helm releases", func() {
		var (
			namespace1, namespace2, secretName1, secretName2 string
			release1, release2                               *rspb.Release
			helmReleaseClient1, helmReleaseClient2           *helmStorageDriver.Secrets
		)
		BeforeEach(func() {
			namespace1 = "default"
			namespace2 = "pink"
			helmReleaseClient1 = helmStorageDriver.NewSecrets(NewSecretsClient(k8sClient, namespace1))
			helmReleaseClient2 = helmStorageDriver.NewSecrets(NewSecretsClient(k8sClient, namespace2))
			secretName1, release1 = newUnicorn("punkunicorn", namespace1, "unicorn", "0.0.1", "1.1.0", 1, rspb.StatusDeployed)
			secretName2, release2 = newUnicorn("pinkunicorn", namespace2, "unicorn", "0.0.1", "1.1.0", 1, rspb.StatusDeployed)
		})
		It("exports proper helm_release metrics (./fixtures/two_releases.metrics)", func() {
			By("Creating helm release 1")
			Expect(helmReleaseClient1.Create(secretName1, release1)).Should(Succeed())
			By("Creating a second namespace")
			Expect(k8sClient.Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace2,
				},
			})).To(Succeed())
			By("Creating helm release 2")
			Expect(helmReleaseClient2.Create(secretName2, release2)).Should(Succeed())

			Eventually(func() (int, error) {
				return testutil.GatherAndCount(metrics.Registry, RelevantMetricNames...)
			}, timeout, interval).Should(Equal(16))

			metricsPath := "../fixtures/two_releases.metrics"
			exp, err := os.Open(metricsPath)
			Expect(err).NotTo(HaveOccurred())
			defer exp.Close()

			// I don't know why exactly but without checking with GatherAndCount first this constantly
			// fails even with increased timeout/internal.
			Eventually(func() error {
				return testutil.GatherAndCompare(metrics.Registry, exp, RelevantMetricNames...)
			}, timeout, interval).Should(Succeed())
		})
		It("does no longer export metrics for a deleted helm release", func() {
			helmReleaseClient1.Delete(secretName1)
			Eventually(func() (int, error) {
				return testutil.GatherAndCount(metrics.Registry, RelevantMetricNames...)
			}, timeout, interval).Should(Equal(8))
		})
	})
	When("There are multiple helm release revisions", func() {
		var (
			namespace, secretName1, secretName2, secretName3 string
			release1, release2, release3                     *rspb.Release
			helmReleaseClient                                *helmStorageDriver.Secrets
			exp                                              io.Reader
		)
		BeforeEach(func() {
			namespace = "default"
			helmReleaseClient = helmStorageDriver.NewSecrets(NewSecretsClient(k8sClient, namespace))
			secretName1, release1 = newUnicorn("punkunicorn", namespace, "unicorn", "0.0.1", "1.1.0", 1, rspb.StatusDeployed)
			secretName2, release2 = newUnicorn("punkunicorn", namespace, "unicorn", "0.0.2", "1.1.0", 2, rspb.StatusDeployed)
			secretName3, release3 = newUnicorn("punkunicorn", namespace, "unicorn", "0.0.3", "1.1.0", 3, rspb.StatusDeployed)

			metricsPath := "../fixtures/multiple_revisions.metrics"
			fixture, err := os.ReadFile(metricsPath)
			Expect(err).NotTo(HaveOccurred())
			exp = bytes.NewReader(fixture)
		})
		It("exports proper helm_release metrics (./fixtures/multiple_revisions.metrics)", func() {
			By("Creating helm releases")
			Expect(helmReleaseClient.Create(secretName1, release1)).Should(Succeed())
			Expect(helmReleaseClient.Create(secretName2, release2)).Should(Succeed())
			Expect(helmReleaseClient.Create(secretName3, release3)).Should(Succeed())

			Eventually(func() (int, error) {
				return testutil.GatherAndCount(metrics.Registry, RelevantMetricNames...)
			}, timeout, interval).Should(Equal(16))

			// I don't know why exactly but without checking with GatherAndCount first this constantly
			// fails even with increased timeout/internal.
			Eventually(func() error {
				return testutil.GatherAndCompare(metrics.Registry, exp, RelevantMetricNames...)
			}, timeout, interval).Should(Succeed())
		})
		It("keeps exporting the same metrics when a historic revision gets deleted", func() {
			helmReleaseClient.Delete(secretName1)
			Eventually(func() error {
				return testutil.GatherAndCompare(metrics.Registry, exp, RelevantMetricNames...)
			}, timeout, interval).Should(Succeed())
		})
	})
})
