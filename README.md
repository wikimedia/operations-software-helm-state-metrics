# helm-state-metrics

helm-state-metrics is a kubernetes controller collecting information about all helm releases in a cluster.

The following metrics are generated for a failed release "foo" of chart "chickadee" in namespace "default":
```
# HELP helm_release_info Information about helm release
# TYPE helm_release_info gauge
helm_release_info{app_version="0.0.1",chart="chickadee",chart_version="1.0.0",name="foo",namespace="default", revision="4"} 1
# HELP helm_release_revision Currently deployed helm chart revision
# TYPE helm_release_revision gauge
helm_release_revision{name="foo",namespace="default"} 4
# HELP helm_release_status Status of a helm release
# TYPE helm_release_status gauge
helm_release_status{name="foo",namespace="default",status="deployed"} 0
helm_release_status{name="foo",namespace="default",status="failed"} 1
helm_release_status{name="foo",namespace="default",status="pending-install"} 0
helm_release_status{name="foo",namespace="default",status="pending-rollback"} 0
helm_release_status{name="foo",namespace="default",status="pending-upgrade"} 0
# HELP helm_release_updated Release update Unix time
# TYPE helm_release_updated gauge
helm_release_updated{name="foo",namespace="default"} 1e+09
```
## How it works
Helm 3 stores information about each helm release (like its state as well as all chart templates, the releases values and the actual rendered manifest) in Kubernetes Secret objects of type `helm.sh/release.v1` within the Namespace of the release (use `kubectl get secrets --field-selector type=helm.sh/release.v1` to take a look).

helm-state-metrics reads fetches those secrets from the Kubernetes API, decodes the content and exports details in Prometheus format.

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources (the Secret objects).

`main.go` contains code for command line handling as well as initializing the controller ([manager](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Manager)) with the [reconciler](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile#Reconciler) (SecretReconciler).

`controllers/secret_controller.go` contains the primary logic inside of the `Reconcile` function which will be called for every change to a relevant Secret object. Fetching the Secret from the Kubernetes API, decoding it and collecting metrics is done here.

`controllers/helm_metrics.go` initializes and registers the prometheus metrics to be exported.

`controllers/secret_interface.go` contains a helper that implements the [SecretsInterface](https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#SecretInterface) the helm library expects to use to interact with the Kubernetes API.

`controllers/utils.go` contains some helper functions, mostly from the helm source code (because they are not exported) to decode information from the Secret objects.


## Historical context
helm-state-metrics started out as a simple prometheus collector iterating over all Helm Secret object on every scrape. This turned out to increase the tail latency of LIST calls for secrets to the Kubernetes API by quite a bit, getting more and more worse with more releases/revisions being added to the cluster.
With the controller approach those "full scans" should happen less often (on restart or periodical full reconciliation only), taking the burden off the Kubernetes API server.