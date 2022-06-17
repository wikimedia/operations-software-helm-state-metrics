package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/namsral/flag"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

var (
	Version       = "development"
	GoVersion     = runtime.Version()
	listenAddress = flag.String("listen-address", ":9104", "Address on which to expose metrics")
	debug         = flag.Bool("debug", false, "Run in debug mode")
	version       = flag.Bool("version", false, "Print version number and exit")
	settings      = cli.New()
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func debugLog(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Version: \"%s\", GoVersion: \"%s\"\n", Version, GoVersion)
		return
	}

	settings.Debug = *debug
	cfg := new(action.Configuration)
	// init k8s rest client for all namespaces ("")
	if err := cfg.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		log.Fatalln(err)
	}

	hc := NewHelmCollector(cfg)
	prometheus.MustRegister(hc)
	http.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatalln(err)
	}

}
