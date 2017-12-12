package httpapi

import (
	"fmt"
	"net/http"
)

type Handler struct {
	metrics map[string]float64
}

func NewHandler() *Handler {
	return &Handler{
		metrics: map[string]float64{},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/metrics" {
		h.handleMetrics(w, r)
	} else {
		w.WriteHeader(404)
		fmt.Fprintln(w, "404 not found")
	}
}

func (h *Handler) UpdateMetric(k string, v float64) {
	h.metrics[k] = v
}

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	for k, v := range h.metrics {
		fmt.Fprintf(w, "spotscaler_%s{} %f\n", k, v)
	}
}
