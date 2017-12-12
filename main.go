// Copyright 2015 The Prometheus Authors
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
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	buildDate string
	commitId  string
	// Business metrics
	cpuTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_temperature_celsius",
		Help: "Current temperature of the CPU.",
	})
	hdFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hd_errors_total",
			Help: "Number of hard-disk errors.",
		},
		[]string{"device"},
	)
	// HTTP handler metrics
	cpuVec     *prometheus.HistogramVec
	hdVec      *prometheus.HistogramVec
	counterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests.",
		},
		[]string{"method", "code"},
	)

	hdDevices = []string{"sda", "sdb"}
)

func init() {
	histogramOpts := prometheus.HistogramOpts{
		Name:        "http_request_duration_seconds",
		Help:        "Histogram of latencies for HTTP requests.",
		Buckets:     []float64{.25, .5, 1, 2.5, 5, 10},
		ConstLabels: prometheus.Labels{"handler": "cpu"},
	}
	cpuVec = prometheus.NewHistogramVec(
		histogramOpts,
		[]string{"method"},
	)
	histogramOpts.ConstLabels = prometheus.Labels{"handler": "hd"}
	hdVec = prometheus.NewHistogramVec(
		histogramOpts,
		[]string{"method"},
	)

	prometheus.MustRegister(cpuTemp)
	prometheus.MustRegister(hdFailures)
	prometheus.MustRegister(cpuVec)
	prometheus.MustRegister(hdVec)
	prometheus.MustRegister(counterVec)
	if buildDate != "" && commitId != "" {
		version := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "version_info",
			Help:        "Information about the app version.",
			ConstLabels: prometheus.Labels{"build_date": buildDate, "commit_id": commitId},
		})
		version.Set(1)
		prometheus.MustRegister(version)
	}
}

type Server struct {
	username string
	password string
}

func (s *Server) auth(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if s.username != "" && s.password != "" && (user != s.username || pass != s.password || !ok) {
			http.Error(w, "Unauthorized.", 401)
			return
		}
		fn(w, r)
	}
}

func (s *Server) cpuHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		m := &dto.Metric{}
		if cpuTemp.Write(m) == nil {
			io.WriteString(w, fmt.Sprintf("The cpu temperature is %.2f°C\n", m.GetGauge().GetValue()))
		}
	} else if r.Method == http.MethodPost {
		s, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error processing request", http.StatusInternalServerError)
		}
		if v, err := strconv.ParseFloat(string(s), 64); err == nil {
			cpuTemp.Set(v)
		} else {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) hdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		for _, d := range hdDevices {
			m := &dto.Metric{}
			if hdFailures.WithLabelValues(d).Write(m) == nil {
				io.WriteString(w,
					fmt.Sprintf("The number of failures for %v is %.0f\n", d, m.GetCounter().GetValue()))
			}
		}
	} else if r.Method == http.MethodPost {
		s, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error processing request", http.StatusInternalServerError)
		}
		for _, d := range hdDevices {
			if string(s) == d {
				hdFailures.WithLabelValues(d).Inc()
				return
			}
		}
		http.Error(w, "Invalid request", http.StatusBadRequest)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ready")
}

func (s *Server) healthyHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Healthy")
}

func main() {
	var listen = kingpin.Flag("listen", "Listen address").Default("127.0.0.1:8080").String()
	var listenm = kingpin.Flag("listen-metrics", "Listen address for exposing metrics (default to 'listen' if blank)").Default("").String()
	var auth = kingpin.Flag("basic-auth", "Basic authentication (eg <user>:<password>)").Default("").String()
	kingpin.Parse()

	var s *Server
	userpass := strings.SplitN(*auth, ":", 2)
	if len(userpass) == 2 {
		log.Println("Basic authentication enabled")
		s = &Server{username: userpass[0], password: userpass[1]}
	} else {
		s = &Server{}
	}

	cpuTemp.Set(37.0)
	for _, d := range hdDevices {
		hdFailures.WithLabelValues(d)
	}

	http.Handle("/-/healthy", http.HandlerFunc(s.readyHandler))
	http.Handle("/-/ready", http.HandlerFunc(s.healthyHandler))
	http.Handle("/cpu",
		promhttp.InstrumentHandlerCounter(counterVec,
			promhttp.InstrumentHandlerDuration(cpuVec,
				s.auth(http.HandlerFunc(s.cpuHandler)),
			),
		),
	)
	http.Handle("/hd",
		promhttp.InstrumentHandlerCounter(counterVec,
			promhttp.InstrumentHandlerDuration(hdVec,
				s.auth(http.HandlerFunc(s.hdHandler)),
			),
		),
	)

	if *listenm != "" {
		mux := http.NewServeMux()
		mux.Handle("/metrics", s.auth(promhttp.Handler().ServeHTTP))
		log.Println("Listening on", *listenm, "(metrics only)")
		go func() {
			log.Fatal(http.ListenAndServe(*listenm, mux))
		}()
	} else {
		http.Handle("/metrics",
			promhttp.InstrumentHandlerCounter(counterVec,
				s.auth(promhttp.Handler().ServeHTTP),
			),
		)
	}

	log.Println("Listening on", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}
