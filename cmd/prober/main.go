package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	probeSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "url_shortener_probe_success",
		Help: "Latest probe result: 1 = success, 0 = failure",
	})

	probeDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "url_shortener_probe_duration_seconds",
		Help: "Duration of the latest probe",
	})

	probeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "url_shortener_probe_duration_seconds_histogram",
		Help:    "Probe duration distribution",
		Buckets: prometheus.DefBuckets,
	})

	probeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_shortener_probe_total",
			Help: "Total number of probes",
		},
		[]string{"result"},
	)
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func recordProbe(success bool, duration time.Duration) {
	probeDuration.Set(duration.Seconds())
	probeHistogram.Observe(duration.Seconds())

	if success {
		probeSuccess.Set(1)
		probeTotal.WithLabelValues("success").Inc()
	} else {
		probeSuccess.Set(0)
		probeTotal.WithLabelValues("failure").Inc()
	}
}

func checkURL(client *http.Client, target string) {
	start := time.Now()

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		recordProbe(false, time.Since(start))
		log.Println("Request error:", err)
		return
	}

	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		recordProbe(false, duration)
		log.Println("Probe failed:", err)
		return
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	recordProbe(success, duration)

	log.Printf(
		"Probe: status=%d success=%t duration=%s",
		resp.StatusCode,
		success,
		duration,
	)
}

func main() {
	prometheus.MustRegister(
		probeSuccess,
		probeDuration,
		probeHistogram,
		probeTotal,
	)

	target := env(
		"PROBE_URL",
		"http://url-shortener.url-shortener.svc.cluster.local:8080/health",
	)

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              ":9091",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		log.Println("Metrics server listening on :9091")

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	checkURL(client, target)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

loop:
	for {
		select {
		case <-ticker.C:
			checkURL(client, target)

		case <-stop:
			break loop
		}
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Shutdown error:", err)
	}

	wg.Wait()
}
