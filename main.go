// Copyright 2018 Simon Pasquier
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/simonpasquier/instrumented_app/version"
)

var (
	stages = []string{"validation", "payment", "shipping"}
	// (fake) business metrics
	sessions = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "user_sessions",
		Help: "Current number of user sessions.",
	})
	errOrders = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_errors_total",
			Help: "Total number of order errors (slow moving counter).",
		},
		[]string{"stage"},
	)
	orders = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_total",
			Help: "Total number of orders (fast moving counter).",
		},
	)
	// HTTP handler metrics
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of latencies for HTTP requests.",
			Buckets: []float64{.05, .1, .5, 1, 2.5, 5, 10},
		},
		[]string{"handler", "method", "code"},
	)
	reqSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Histogram of response sizes for HTTP requests.",
			Buckets: []float64{1024, 2048, 4096, 16384, 65536, 131072},
		},
		[]string{"handler", "method", "code"},
	)
)

func init() {
	prometheus.MustRegister(errOrders)
	prometheus.MustRegister(orders)
	prometheus.MustRegister(sessions)
	prometheus.MustRegister(reqDuration)
	prometheus.MustRegister(reqSize)
	if version.BuildDate != "" && version.Revision != "" {
		version := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "instrumented_app_info",
			Help:        "Information about the app version.",
			ConstLabels: prometheus.Labels{"build_date": version.BuildDate, "version": version.Revision},
		})
		version.Set(1)
		prometheus.MustRegister(version)
	}
}

type server struct {
	username string
	password string
}

func updateMetrics(d time.Duration, done chan struct{}) {
	t := time.NewTimer(d)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		select {
		case <-t.C:
			sessions.Set(float64(r.Intn(100)))
			orders.Add(float64(r.Intn(5)))
			for _, s := range stages {
				errOrders.WithLabelValues(s).Add(math.Floor(float64(r.Intn(100)) / 97))
			}
			t.Reset(d)
		case <-done:
			return
		}
	}
}

func (s *server) auth(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if s.username != "" && s.password != "" && (user != s.username || pass != s.password || !ok) {
			http.Error(w, "Unauthorized.", 401)
			return
		}
		fn(w, r)
	}
}

func (s *server) readyHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ready")
}

func (s *server) healthyHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Healthy")
}

func (s *server) helloHandler(w http.ResponseWriter, _ *http.Request) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(int64(r.Intn(1000))) * time.Millisecond)
	io.WriteString(w, "Hello!")
}

func registerHandler(handleFunc func(pattern string, handler http.Handler), name string, handler http.HandlerFunc) {
	handleFunc(name, promhttp.InstrumentHandlerDuration(
		reqDuration.MustCurryWith(prometheus.Labels{"handler": name}),
		promhttp.InstrumentHandlerResponseSize(
			reqSize.MustCurryWith(prometheus.Labels{"handler": name}),
			handler,
		),
	))
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Demo application for Prometheus instrumentation.")
	app.Version(version.Revision)
	app.HelpFlag.Short('h')

	var listen = app.Flag("listen", "Listen address").Default("127.0.0.1:8080").String()
	var listenm = app.Flag("listen-metrics", "Listen address for exposing metrics (default to 'listen' if blank)").Default("").String()
	var auth = app.Flag("basic-auth", "Basic authentication (eg <user>:<password>)").Default("").String()

	_, err := app.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("%s, try --help", err)
	}

	var s *server
	userpass := strings.SplitN(*auth, ":", 2)
	if len(userpass) == 2 {
		log.Println("Basic authentication enabled")
		s = &server{username: userpass[0], password: userpass[1]}
	} else {
		s = &server{}
	}

	var wg sync.WaitGroup
	done := make(chan struct{})
	go func() {
		wg.Add(1)
		defer wg.Done()
		updateMetrics(time.Duration(time.Second), done)
	}()

	registerHandler(http.Handle, "/-/healthy", http.HandlerFunc(s.readyHandler))
	registerHandler(http.Handle, "/-/ready", http.HandlerFunc(s.readyHandler))
	registerHandler(http.Handle, "/", s.auth(http.HandlerFunc(s.helloHandler)))

	if *listenm != "" {
		mux := http.NewServeMux()
		registerHandler(mux.Handle, "/metrics", s.auth(promhttp.Handler().ServeHTTP))
		go func() {
			wg.Add(1)
			defer wg.Done()
			log.Println("Listening on", *listenm, "(metrics only)")
			log.Fatal(http.ListenAndServe(*listenm, mux))
		}()
	} else {
		log.Println("Registering /metrics")
		registerHandler(http.Handle, "/metrics", s.auth(promhttp.Handler().ServeHTTP))
	}

	go func() {
		wg.Add(1)
		defer wg.Done()
		log.Println("Listening on", *listen)
		log.Fatal(http.ListenAndServe(*listen, nil))
	}()

	wg.Wait()
}
