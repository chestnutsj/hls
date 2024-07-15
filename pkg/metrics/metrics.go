package metrics

import (
	"github.com/chestnutsj/hls/pkg/tools"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
	"net/http/pprof" // 导入pprof包，使其HTTP服务可用
	"strconv"
)

var startTime prometheus.Gauge

func init() {
	startTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "demo",
		Subsystem: tools.AppName(),
		Name:      "start_time",
		Help:      "this app start time",
	})
	prometheus.MustRegister(startTime)
}

func StartMetrics(port string, debug bool) {
	address := ":" + port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	log.Println("Using Metrics port:", listener.Addr().(*net.TCPAddr).Port)

	mux := http.NewServeMux()
	if debug {

		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	mux.Handle("/metrics", promhttp.Handler())
	log.Println("localhost:" + strconv.Itoa(listener.Addr().(*net.TCPAddr).Port) + "/metrics")
	startTime.SetToCurrentTime()

	panic(http.Serve(listener, mux))
}
