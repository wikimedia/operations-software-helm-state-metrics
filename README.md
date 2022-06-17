# helm-state-metrics

helm-state-metrics is a prometheus exporter collecting information about all helm releases in a cluster.

The following metrics are generated for a failed release "foo" of chart "chickadee" in namespace "default":
```
# HELP helm_release_info Information about helm release
# TYPE helm_release_info gauge
helm_release_info{app_version="0.0.1",chart="chickadee",chart_version="1.0.0",name="foo",namespace="default"} 1
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