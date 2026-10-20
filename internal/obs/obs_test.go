package obs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetricsPreservesStreaming(t *testing.T) {
	h := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush: %v", err)
		}
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestMetricsBoundedRouteLabels(t *testing.T) {
	HttpRequestsTotal.Reset()
	HttpRequestDurationSeconds.Reset()
	r := chi.NewRouter()
	r.Use(MetricsMiddleware)
	r.Get("/bots/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/bots/secret-token", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/unknown-unique-path", nil))
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, family := range metrics {
		if family.GetName() == "bottagon_http_requests_total" {
			for _, metric := range family.Metric {
				for _, label := range metric.Label {
					if label.GetName() == "path" {
						paths[label.GetValue()] = true
					}
				}
			}
		}
	}
	if len(paths) != 2 || !paths["/bots/{id}"] || !paths["unmatched"] {
		t.Fatalf("unbounded labels: %v", paths)
	}
}

func TestMetricsRecordsFirstStatus(t *testing.T) {
	HttpRequestsTotal.Reset()
	h := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); w.WriteHeader(500) }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	metric := &dto.Metric{}
	if err := HttpRequestsTotal.WithLabelValues("GET", "unmatched", "Created").Write(metric); err != nil {
		t.Fatal(err)
	}
	if got := metric.GetCounter().GetValue(); got != 1 {
		t.Fatalf("got %v", got)
	}
}
